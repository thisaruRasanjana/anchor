package main

import "time"

type Store struct {
	data map[string]Entry
}

type Entry struct {
	Value     string
	ExpiresAt time.Time
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

func (s *Store) Set(key string, value string) {
	s.data[key] = Entry{Value: value}
}

func (s *Store) Get(key string) (string, bool) {
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

func (s *Store) Delete(key string) {
	delete(s.data, key)
}

func (s *Store) SetWithTTL(key string, value string, ttl int) {
	expirationTime := time.Now().Add(time.Duration(ttl) * time.Second)
	s.data[key] = Entry{Value: value, ExpiresAt: expirationTime}
}
