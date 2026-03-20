package main

type Store struct {
	data map[string]string
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key string, value string) {
	s.data[key] = value
}

func (s *Store) Get(key string) (string, bool) {
	value, ok := s.data[key]
	return value, ok
}

func (s *Store) Delete(key string) {
	delete(s.data, key)
}