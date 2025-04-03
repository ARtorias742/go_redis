package store

import (
	"sync"
	"time"
)

type Entry struct {
	value     string
	expiresAt int64 // Unix timestamp in milliseconds, 0 if no expiration
}

type Store struct {
	data map[string]Entry
	mu   sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Entry{value: value, expiresAt: 0}
}

func (s *Store) SetWithExpiry(key, value string, ttl int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt := time.Now().UnixMilli() + ttl*1000
	s.data[key] = Entry{value: value, expiresAt: expiresAt}

}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.Unlock()
	entry, exists := s.data[key]
	if !exists {
		return "", false
	}
	if entry.expiresAt > 0 && time.Now().UnixMilli() > entry.expiresAt {
		delete(s.data, key)
		return "", false
	}

	return entry.value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		return true
	}
	return false
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.data[key]
	if exists && s.data[key].expiresAt > 0 && time.Now().UnixMilli() > s.data[key].expiresAt {
		delete(s.data, key)
		return false
	}

	return exists
}
