package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDatabaseSet(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Thisaru"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, ok := store.Get("name")
	if !ok {
		t.Fatal("expected name to exist")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %q", value)
	}
}

func TestDatabaseDelete(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Thisaru"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	deleted, err := db.Delete("name")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if !deleted {
		t.Fatal("expected Delete to return true")
	}

	_, ok := store.Get("name")
	if ok {
		t.Fatal("expected name to be deleted")
	}
}

func TestDatabaseSetTTL(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.SetTTL("token", "abc123", 10); err != nil {
		t.Fatalf("SetTTL failed: %v", err)
	}

	value, ok := store.Get("token")
	if !ok {
		t.Fatal("expected token to exist")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %q", value)
	}
}

func TestDatabasePersistence(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Thisaru"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	wal.Close()

	recoveredWAL, err := NewWAL(dir + "/test.wal")
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := recoveredStore.Get("name")
	if !ok {
		t.Fatal("expected name to be recovered")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %q", value)
	}
}

func TestDatabaseRecoverySetDelete(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Alice"); err != nil {
		t.Fatalf("SET failed: %v", err)
	}

	if _, err := db.Delete("name"); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	wal.Close()

	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	_, ok := recoveredStore.Get("name")

	if ok {
		t.Fatal("expected name to be absent after recovery")
	}
}

func TestDatabaseRecoveryLatestSetWins(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Alice"); err != nil {
		t.Fatalf("first SET failed: %v", err)
	}

	if err := db.Set("name", "Bob"); err != nil {
		t.Fatalf("second SET failed: %v", err)
	}

	wal.Close()

	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := recoveredStore.Get("name")

	if !ok {
		t.Fatal("expected name to exist")
	}

	if value != "Bob" {
		t.Fatalf("expected Bob, got %q", value)
	}
}

func TestDatabaseRecoverySetDeleteSet(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.Set("name", "Alice"); err != nil {
		t.Fatalf("first SET failed: %v", err)
	}

	if _, err := db.Delete("name"); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	if err := db.Set("name", "Bob"); err != nil {
		t.Fatalf("second SET failed: %v", err)
	}

	wal.Close()

	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := recoveredStore.Get("name")

	if !ok {
		t.Fatal("expected name to exist")
	}

	if value != "Bob" {
		t.Fatalf("expected Bob, got %q", value)
	}
}

func TestDatabaseRecoveryValidTTL(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.SetTTL("token", "abc123", 10); err != nil {
		t.Fatalf("SETTTL failed: %v", err)
	}

	wal.Close()

	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := recoveredStore.Get("token")

	if !ok {
		t.Fatal("expected token to exist after recovery")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %q", value)
	}
}

func TestDatabaseRecoveryExpiredTTL(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	if err := db.SetTTL("token", "abc123", 1); err != nil {
		t.Fatalf("SETTTL failed: %v", err)
	}

	wal.Close()

	time.Sleep(1100 * time.Millisecond)

	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	_, ok := recoveredStore.Get("token")

	if ok {
		t.Fatal("expected expired token to be absent after recovery")
	}
}

func TestDatabaseConcurrentWorkloadRecovery(t *testing.T) {
	const clients = 20
	const operationsPerClient = 50

	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	store := NewStore()
	db := NewDatabase(store, wal)

	var wg sync.WaitGroup

	for i := 0; i < clients; i++ {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			for j := 0; j < operationsPerClient; j++ {
				key := fmt.Sprintf("client%d-key%d", clientID, j)
				value := fmt.Sprintf("value-%d", j)

				switch j % 4 {
				case 0:
					if err := db.Set(key, value); err != nil {
						t.Errorf("SET failed: %v", err)
					}

				case 1:
					if err := db.Set(key, value); err != nil {
						t.Errorf("SET failed: %v", err)
					}

					_, ok := db.Get(key)
					if !ok {
						t.Errorf("expected key %q to exist", key)
					}

				case 2:
					if err := db.Set(key, value); err != nil {
						t.Errorf("SET failed: %v", err)
					}

					if _, err := db.Delete(key); err != nil {
						t.Errorf("DELETE failed: %v", err)
					}

				case 3:
					if err := db.SetTTL(key, value, 60); err != nil {
						t.Errorf("SETTTL failed: %v", err)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Close the WAL to simulate the end of the original process.
	if err := wal.Close(); err != nil {
		t.Fatalf("failed to close WAL: %v", err)
	}

	// Recover into a completely fresh Store.
	recoveredWAL, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer recoveredWAL.Close()

	recoveredStore := NewStore()

	if err := recoveredWAL.Replay(recoveredStore); err != nil {
		t.Fatalf("recovery failed: %v", err)
	}

	// Compare the expected final state with the recovered state.
	for i := 0; i < clients; i++ {
		for j := 0; j < operationsPerClient; j++ {
			key := fmt.Sprintf("client%d-key%d", i, j)

			originalValue, originalExists := store.Get(key)
			recoveredValue, recoveredExists := recoveredStore.Get(key)

			if originalExists != recoveredExists {
				t.Errorf(
					"key %q: original exists=%v, recovered exists=%v",
					key,
					originalExists,
					recoveredExists,
				)
				continue
			}

			if originalExists && originalValue != recoveredValue {
				t.Errorf(
					"key %q: original=%q, recovered=%q",
					key,
					originalValue,
					recoveredValue,
				)
			}
		}
	}
}
