package main

import (
	"fmt"
	"sync"
	"time"
)

type Database struct {
	store *Store
	wal   *WAL
	mu    sync.Mutex
}

func NewDatabase(store *Store, wal *WAL) *Database {
	return &Database{
		store: store,
		wal:   wal,
	}
}

func (db *Database) Set(key, value string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	record := "SET " + key + " " + value

	if err := db.wal.Append(record); err != nil {
		return err
	}

	db.store.Set(key, value)

	return nil
}

func (db *Database) Delete(key string) (bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	record := "DELETE " + key

	if err := db.wal.Append(record); err != nil {
		return false, err
	}

	deleted := db.store.Delete(key)

	return deleted, nil
}

func (db *Database) SetTTL(key, value string, ttl int) error {
	if ttl <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)

	record := fmt.Sprintf(
		"SET_EXPIRED %s %s %s",
		key,
		value,
		expiresAt.Format(time.RFC3339Nano),
	)

	if err := db.wal.Append(record); err != nil {
		return err
	}

	db.store.SetWithExpiration(key, value, expiresAt)

	return nil
}

func (db *Database) Get(key string) (string, bool) {
	return db.store.Get(key)
}
