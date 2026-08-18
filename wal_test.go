package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWALAppend(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	err = wal.Append("SET name Thisaru")
	if err != nil {
		t.Fatalf("failed to append to WAL: %v", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read WAL: %v", err)
	}

	if !strings.Contains(string(data), "SET name Thisaru\n") {
		t.Fatalf("expected WAL to contain SET record, got %q", string(data))
	}
}

func TestWALReplay(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	if err := wal.Append("SET name Thisaru"); err != nil {
		t.Fatal(err)
	}

	if err := wal.Append("SET age 22"); err != nil {
		t.Fatal(err)
	}

	if err := wal.Append("DELETE age"); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	if err := wal.Replay(store); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := store.Get("name")
	if !ok || value != "Thisaru" {
		t.Fatalf("expected name=Thisaru, got %q, exists=%v", value, ok)
	}

	_, ok = store.Get("age")
	if ok {
		t.Fatal("expected age to be deleted after replay")
	}
}

func TestWALReplayTTL(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	future := time.Now().Add(10 * time.Second)
	past := time.Now().Add(-10 * time.Second)

	futureRecord := fmt.Sprintf(
		"SET_EXPIRED valid value %s",
		future.Format(time.RFC3339Nano),
	)

	pastRecord := fmt.Sprintf(
		"SET_EXPIRED expired value %s",
		past.Format(time.RFC3339Nano),
	)

	if err := wal.Append(futureRecord); err != nil {
		t.Fatal(err)
	}

	if err := wal.Append(pastRecord); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	if err := wal.Replay(store); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	value, ok := store.Get("valid")
	if !ok {
		t.Fatal("expected valid key to be restored")
	}

	if value != "value" {
		t.Fatalf("expected value, got %q", value)
	}

	_, ok = store.Get("expired")
	if ok {
		t.Fatal("expected expired key not to be restored")
	}
}

func TestWALReplayUnknownCommand(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	if err := wal.Append("BOGUS something"); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	err = wal.Replay(store)
	if err == nil {
		t.Fatal("expected replay to fail for unknown command")
	}
}

func TestWALReplayMalformedSet(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	if err := wal.Append("SET onlykey"); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	err = wal.Replay(store)
	if err == nil {
		t.Fatal("expected replay to fail for malformed SET")
	}
}

func TestWALReplayInvalidExpiration(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	if err := wal.Append("SET_EXPIRED token abc123 definitely-not-a-time"); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	err = wal.Replay(store)
	if err == nil {
		t.Fatal("expected replay to fail for invalid expiration timestamp")
	}
}

func TestWALConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			entry := fmt.Sprintf("SET key%d value%d", i, i)

			if err := wal.Append(entry); err != nil {
				t.Errorf("append failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read WAL: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) != 100 {
		t.Fatalf("expected 100 WAL records, got %d", len(lines))
	}
}

func TestWALReplayIncompleteFinalRecord(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	_, err = file.WriteString(
		"SET name Thisaru\n" +
			"SET age 22\n" +
			"SET broken",
	)
	if err != nil {
		t.Fatalf("failed to write WAL: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close WAL: %v", err)
	}

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()

	err = wal.Replay(store)

	if err != nil {
		t.Fatalf("expected incomplete final record to be ignored, got: %v", err)
	}

	value, ok := store.Get("name")
	if !ok {
		t.Fatal("expected name to be recovered")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %q", value)
	}

	value, ok = store.Get("age")
	if !ok {
		t.Fatal("expected age to be recovered")
	}

	if value != "22" {
		t.Fatalf("expected 22, got %q", value)
	}

	_, ok = store.Get("broken")
	if ok {
		t.Fatal("incomplete final record should not be recovered")
	}
}

func TestWALReplayMalformedCompleteRecord(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.wal"

	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	_, err = file.WriteString(
		"SET name Thisaru\n" +
			"SET broken\n",
	)
	if err != nil {
		t.Fatalf("failed to write WAL: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close WAL: %v", err)
	}

	wal, err := NewWAL(filename)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	store := NewStore()

	err = wal.Replay(store)

	if err == nil {
		t.Fatal("expected replay to fail on malformed complete record")
	}
}
