package main
import(
	"bufio"
	"fmt"
	"os"
	"strings"
)

type WAL struct{
	file *os.File
}

func NewWAL(filename string) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{file: file}, nil
}

func (wal *WAL) Append(entry string) error {
	_, err := wal.file.WriteString(entry + "\n")
	if err != nil {
		return err
	}
	return wal.file.Sync()
}

func (wal *WAL) Close() error {
	return wal.file.Close()
}

func (wal *WAL) Replay(store *Store) error {
	_, err := wal.file.Seek(0, 0)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(wal.file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "SET":
			if len(parts) != 3 {
				return fmt.Errorf("invalid SET command in WAL: %s", line)
			}
			store.Set(parts[1], parts[2])

		case "DELETE":
    		if len(parts) != 2 {
        		return fmt.Errorf("invalid DELETE command in WAL: %s", line)
    		}
    		store.Delete(parts[1])
		default:
			return fmt.Errorf("unknown command in WAL: %s", line)
		}
	}
	return scanner.Err()
}