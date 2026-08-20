package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestStoreSetAndGet(t *testing.T) {
	store := NewStore()

	store.Set("name", "Thisaru")

	value, ok := store.Get("name")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %s", value)
	}
}

func TestStoreGetMissingKey(t *testing.T) {
	store := NewStore()

	value, ok := store.Get("missing")

	if ok {
		t.Fatal("expected key not to exist")
	}

	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}
}

func TestStoreDelete(t *testing.T) {
	store := NewStore()

	store.Set("name", "Thisaru")

	deleted := store.Delete("name")

	if !deleted {
		t.Fatal("expected Delete to return true")
	}

	_, ok := store.Get("name")

	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestStoreDeleteMissingKey(t *testing.T) {
	store := NewStore()

	deleted := store.Delete("missing")

	if deleted {
		t.Fatal("expected Delete to return false")
	}
}

func TestStoreTTL(t *testing.T) {
	store := NewStore()

	store.SetWithTTL("token", "abc123", 1)

	value, ok := store.Get("token")

	if !ok {
		t.Fatal("expected key to exist before expiration")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %s", value)
	}
}

func TestStoreTTLExpiration(t *testing.T) {
	store := NewStore()

	store.SetWithTTL("token", "abc123", 1)

	time.Sleep(1100 * time.Millisecond)

	_, ok := store.Get("token")

	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestStoreSetWithExpiration(t *testing.T) {
	store := NewStore()

	expiresAt := time.Now().Add(1 * time.Second)

	store.SetWithExpiration("token", "abc123", expiresAt)

	value, ok := store.Get("token")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %s", value)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)

			store.Set(key, value)

			got, ok := store.Get(key)
			if !ok {
				t.Errorf("expected %s to exist", key)
				return
			}

			if got != value {
				t.Errorf("expected %s, got %s", value, got)
			}
		}(i)
	}

	wg.Wait()
}

func TestServerConcurrentMixedOperations(t *testing.T) {
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

				switch j % 4 {
				case 0:
					response := sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("SET %s %s", key, value),
					)

					if response != "OK" {
						t.Errorf(
							"SET %s: expected OK, got %q",
							key,
							response,
						)
					}

				case 1:
					sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("SET %s %s", key, value),
					)

					response := sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("GET %s", key),
					)

					if response != value {
						t.Errorf(
							"GET %s: expected %q, got %q",
							key,
							value,
							response,
						)
					}

				case 2:
					sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("SET %s %s", key, value),
					)

					response := sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("DELETE %s", key),
					)

					if response != "OK" {
						t.Errorf(
							"DELETE %s: expected OK, got %q",
							key,
							response,
						)
					}

				case 3:
					response := sendCommand(
						t,
						reader,
						clientConn,
						fmt.Sprintf("SETTTL %s %s 10", key, value),
					)

					if response != "OK" {
						t.Errorf(
							"SETTTL %s: expected OK, got %q",
							key,
							response,
						)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify keys whose final operation was SET.
	for i := 0; i < clients; i++ {
		for j := 0; j < operationsPerClient; j++ {
			key := fmt.Sprintf("client%d-key%d", i, j)

			switch j % 4 {
			case 0, 1, 3:
				value, ok := db.Get(key)

				if !ok {
					t.Errorf("expected key %q to exist", key)
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

			case 2:
				_, ok := db.Get(key)

				if ok {
					t.Errorf("expected key %q to be deleted", key)
				}
			}
		}
	}
}
