package main

import (
	"fmt"
	"net"
	"bufio"
	"strings"
	"strconv"
)

func StartServer(store *Store, wal *WAL) error {
	listener, err := net.Listen("tcp", ":7379")
	if err != nil {
		return fmt.Errorf("failed to listen on :7379: %w", err)
	}
	defer listener.Close()
	fmt.Println("Server listening on :7379")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("failed to accept connection: %v\n", err)
			continue
		}
		go handleConnection(conn, store, wal)
	}
}

func handleConnection(conn net.Conn, store *Store, wal *WAL) {
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
			record := strings.Join(parts, " ")
			if err := wal.Append(record); err != nil {
				fmt.Fprintf(conn, "ERROR: failed to append to WAL: %v\n", err)
				continue
			}
			store.Set(parts[1], parts[2])
			fmt.Fprintln(conn, "OK")

		case "GET":
			if len(parts) != 2 {
				fmt.Fprintln(conn, "ERROR: usage: GET <key>")
				continue
			}
			value, ok := store.Get(parts[1])
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
			deleted := store.Delete(parts[1])
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
			store.SetWithTTL(parts[1], parts[2], ttl)
			fmt.Fprintln(conn, "OK")

		default:
			fmt.Fprintln(conn, "ERROR: unknown command")
		}
	}
}
