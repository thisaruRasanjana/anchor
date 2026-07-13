package main

import (
	"fmt"
	"net"
	"time"
)

func StartServer(store *Store) error {
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected!")
	time.Sleep(10 * time.Second)
	fmt.Println("Client disconnected")
}
