package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Server struct {
	db    *Database
	wg    sync.WaitGroup
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func StartServer(db *Database, stop <-chan os.Signal) error {
	listener, err := net.Listen("tcp", ":7379")
	if err != nil {
		return fmt.Errorf("failed to listen on :7379: %w", err)
	}
	defer listener.Close()

	server := &Server{
		db:    db,
		conns: make(map[net.Conn]struct{}),
	}

	fmt.Println("Server listening on :7379")

	go func() {
		<-stop

		fmt.Println("Shutting down server...")

		listener.Close()

		server.mu.Lock()
		for conn := range server.conns {
			conn.Close()
		}
		server.mu.Unlock()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			fmt.Printf("failed to accept connection: %v\n", err)
			continue
		}

		server.mu.Lock()
		server.conns[conn] = struct{}{}
		server.mu.Unlock()

		server.wg.Add(1)

		go func(conn net.Conn) {
			defer server.wg.Done()

			handleConnection(conn, server.db)

			server.mu.Lock()
			delete(server.conns, conn)
			server.mu.Unlock()
		}(conn)
	}

	server.wg.Wait()

	fmt.Println("Server stopped.")

	return nil
}

func handleConnection(conn net.Conn, db *Database) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		command := scanner.Text()
		parts := strings.Fields(command)

		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "SET":
			if len(parts) != 3 {
				fmt.Fprintln(conn, "ERROR: usage: SET <key> <value>")
				continue
			}

			if err := db.Set(parts[1], parts[2]); err != nil {
				fmt.Fprintf(conn, "ERROR: failed to set key: %v\n", err)
				continue
			}

			fmt.Fprintln(conn, "OK")

		case "GET":
			if len(parts) != 2 {
				fmt.Fprintln(conn, "ERROR: usage: GET <key>")
				continue
			}

			value, ok := db.Get(parts[1])
			if !ok {
				fmt.Fprintln(conn, "NOT FOUND")
			} else {
				fmt.Fprintln(conn, value)
			}

		case "DELETE":
			if len(parts) != 2 {
				fmt.Fprintln(conn, "ERROR: usage: DELETE <key>")
				continue
			}

			deleted, err := db.Delete(parts[1])
			if err != nil {
				fmt.Fprintf(conn, "ERROR: failed to delete key: %v\n", err)
				continue
			}

			if deleted {
				fmt.Fprintln(conn, "OK")
			} else {
				fmt.Fprintln(conn, "NOT FOUND")
			}

		case "SETTTL":
			if len(parts) != 4 {
				fmt.Fprintln(conn, "ERROR: usage: SETTTL <key> <value> <ttl>")
				continue
			}

			ttl, err := strconv.Atoi(parts[3])
			if err != nil {
				fmt.Fprintln(conn, "ERROR: TTL must be an integer")
				continue
			}

			if err := db.SetTTL(parts[1], parts[2], ttl); err != nil {
				fmt.Fprintf(conn, "ERROR: %v\n", err)
				continue
			}

			fmt.Fprintln(conn, "OK")

		default:
			fmt.Fprintln(conn, "ERROR: unknown command")
		}
	}
}
