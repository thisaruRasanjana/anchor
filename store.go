package main

import (
	"sync"
	"time"
)

type Store struct {
	data map[string]Entry
	mu   sync.RWMutex
}

type Entry struct {
	Value     string
	ExpiresAt time.Time
}

func NewStore() *Store {
	s := &Store{
		data: make(map[string]Entry),
	}
	go s.StartExpirationWorker()
	return s
}

func (s *Store) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Entry{Value: value}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[key]
	if !ok {
		return "", false
	}

	if entry.ExpiresAt.IsZero() {
		return entry.Value, true
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(s.data, key)
		return "", false
	}

	return entry.Value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.data[key]
	if exists {
		delete(s.data, key)
	}
	return exists
}

func (s *Store) SetWithTTL(key string, value string, ttl int) {
	expirationTime := time.Now().Add(time.Duration(ttl) * time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Entry{Value: value, ExpiresAt: expirationTime}
}

func (s *Store) StartExpirationWorker() {
	for {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.data {
			if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
		time.Sleep(1 * time.Second)
	}
}

func (s *Store) SetWithExpiration(key, value string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = Entry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}
