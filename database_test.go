package main

import (
	"testing"
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
