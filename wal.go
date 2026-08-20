package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type WAL struct {
	file *os.File
	mu   sync.Mutex
}

func NewWAL(filename string) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{file: file}, nil
}

func (wal *WAL) Append(entry string) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	_, err := wal.file.WriteString(entry + "\n")
	if err != nil {
		return err
	}
	return wal.file.Sync()
}

func (wal *WAL) Close() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	return wal.file.Close()
}

func (wal *WAL) Replay(store *Store) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	_, err := wal.file.Seek(0, 0)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(wal.file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) == 0 && err == io.EOF {
			break
		}
		terminated := strings.HasSuffix(line, "\n")

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		parts := strings.Fields(line)

		if len(parts) == 0 {
			if err == io.EOF {
				break
			}
			continue
		}

		switch parts[0] {
		case "SET":
			if len(parts) != 3 {
				if err == io.EOF && !terminated {
					break
				}
				return fmt.Errorf("invalid SET command in WAL: %s", line)
			}
			store.Set(parts[1], parts[2])

		case "DELETE":
			if len(parts) != 2 {
				if err == io.EOF && !terminated {
					break
				}
				return fmt.Errorf("invalid DELETE command in WAL: %s", line)
			}
			store.Delete(parts[1])

		case "SET_EXPIRED":
			if len(parts) != 4 {
				if err == io.EOF && !terminated {
					break
				}
				return fmt.Errorf("invalid SET_EXPIRED command in WAL: %s", line)
			}

			expiresAt, err := time.Parse(time.RFC3339Nano, parts[3])
			if err != nil {
				return fmt.Errorf("invalid expiration time in WAL: %s", line)
			}

			if time.Now().Before(expiresAt) {
				store.SetWithExpiration(parts[1], parts[2], expiresAt)
			}

		default:
			return fmt.Errorf("unknown command in WAL: %s", line)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}
	return nil
}
