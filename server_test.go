package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestServer(t *testing.T) (net.Conn, net.Conn, *Database) {
	t.Helper()

	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	t.Cleanup(func() {
		wal.Close()
	})

	store := NewStore()
	db := NewDatabase(store, wal)

	serverConn, clientConn := net.Pipe()

	go handleConnection(serverConn, db)

	return clientConn, serverConn, db
}

func sendCommand(t *testing.T, reader *bufio.Reader, conn net.Conn, command string) string {
	t.Helper()

	_, err := conn.Write([]byte(command + "\n"))
	if err != nil {
		t.Fatalf("failed to send command: %v", err)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	return strings.TrimSpace(response)
}

func TestServerSet(t *testing.T) {
	clientConn, serverConn, db := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SET name Thisaru")

	if response != "OK" {
		t.Fatalf("expected OK, got %q", response)
	}

	value, ok := db.Get("name")

	if !ok {
		t.Fatal("expected name to exist")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %q", value)
	}
}

func TestServerGet(t *testing.T) {
	clientConn, serverConn, db := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	if err := db.Set("name", "Thisaru"); err != nil {
		t.Fatalf("failed to set test value: %v", err)
	}

	response := sendCommand(t, reader, clientConn, "GET name")

	if response != "Thisaru" {
		t.Fatalf("expected Thisaru, got %q", response)
	}
}

func TestServerDelete(t *testing.T) {
	clientConn, serverConn, db := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	if err := db.Set("name", "Thisaru"); err != nil {
		t.Fatalf("failed to set test value: %v", err)
	}

	response := sendCommand(t, reader, clientConn, "DELETE name")

	if response != "OK" {
		t.Fatalf("expected OK, got %q", response)
	}

	_, ok := db.Get("name")

	if ok {
		t.Fatal("expected name to be deleted")
	}
}

func TestServerGetMissingKey(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "GET missing")

	if response != "NOT FOUND" {
		t.Fatalf("expected NOT FOUND, got %q", response)
	}
}

func TestServerUnknownCommand(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "HELLO")

	if response != "ERROR: unknown command" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerSetInvalidArguments(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SET name")

	if response != "ERROR: usage: SET <key> <value>" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerGetInvalidArguments(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "GET")

	if response != "ERROR: usage: GET <key>" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerDeleteInvalidArguments(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "DELETE")

	if response != "ERROR: usage: DELETE <key>" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerSetTTLInvalidArguments(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SETTTL token value")

	if response != "ERROR: usage: SETTTL <key> <value> <ttl>" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerSetTTLInvalidInteger(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SETTTL token value abc")

	if response != "ERROR: TTL must be an integer" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerSetTTLNonPositive(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SETTTL token value 0")

	if response != "ERROR: TTL must be greater than 0" {
		t.Fatalf("unexpected response: %q", response)
	}
}

func TestServerMultipleCommands(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SET name Thisaru")
	if response != "OK" {
		t.Fatalf("SET: expected OK, got %q", response)
	}

	response = sendCommand(t, reader, clientConn, "GET name")
	if response != "Thisaru" {
		t.Fatalf("GET: expected Thisaru, got %q", response)
	}

	response = sendCommand(t, reader, clientConn, "DELETE name")
	if response != "OK" {
		t.Fatalf("DELETE: expected OK, got %q", response)
	}

	response = sendCommand(t, reader, clientConn, "GET name")
	if response != "NOT FOUND" {
		t.Fatalf("GET after DELETE: expected NOT FOUND, got %q", response)
	}
}

func TestServerSetTTLExpiration(t *testing.T) {
	clientConn, serverConn, _ := newTestServer(t)

	defer clientConn.Close()
	defer serverConn.Close()

	reader := bufio.NewReader(clientConn)

	response := sendCommand(t, reader, clientConn, "SETTTL token abc123 1")

	if response != "OK" {
		t.Fatalf("expected OK, got %q", response)
	}

	response = sendCommand(t, reader, clientConn, "GET token")

	if response != "abc123" {
		t.Fatalf("expected abc123, got %q", response)
	}

	time.Sleep(1100 * time.Millisecond)

	response = sendCommand(t, reader, clientConn, "GET token")

	if response != "NOT FOUND" {
		t.Fatalf("expected NOT FOUND after expiration, got %q", response)
	}
}

func TestServerConcurrentClients(t *testing.T) {
	const clients = 20
	const operationsPerClient = 50

	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()
	db := NewDatabase(store, wal)

	var wg sync.WaitGroup

	for i := 0; i < clients; i++ {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			go handleConnection(serverConn, db)

			reader := bufio.NewReader(clientConn)

			for j := 0; j < operationsPerClient; j++ {
				key := fmt.Sprintf("client%d-key%d", clientID, j)
				value := fmt.Sprintf("value-%d", j)

				response := sendCommand(
					t,
					reader,
					clientConn,
					fmt.Sprintf("SET %s %s", key, value),
				)

				if response != "OK" {
					t.Errorf(
						"client %d operation %d: expected OK, got %q",
						clientID,
						j,
						response,
					)
				}
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < clients; i++ {
		for j := 0; j < operationsPerClient; j++ {
			key := fmt.Sprintf("client%d-key%d", i, j)

			value, ok := db.Get(key)
			if !ok {
				t.Errorf("missing key %q", key)
				continue
			}

			expected := fmt.Sprintf("value-%d", j)

			if value != expected {
				t.Errorf(
					"key %q: expected %q, got %q",
					key,
					expected,
					value,
				)
			}
		}
	}
}
