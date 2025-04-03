package store

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Store struct {
	data        map[string]Entry
	mu          sync.RWMutex
	aofChan     chan []string
	replicaChan chan []string
}

func NewStore() *Store {
	return &Store{
		data:        make(map[string]Entry),
		aofChan:     make(chan []string, 100),
		replicaChan: make(chan []string, 100),
	}
}

func (s *Store) SetString(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = Entry{Type: TypeString, StringVal: value}
	s.aofChan <- []string{"SET", key, value}
	s.replicaChan <- []string{"SET", key, value}
}

func (s *Store) SetWithExpiry(key, value string, ttl int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt := time.Now().UnixMilli() + ttl*1000
	s.data[key] = Entry{Type: TypeString, StringVal: value, ExpiresAt: expiresAt}
	s.aofChan <- []string{"SET", key, value}
	s.replicaChan <- []string{"SET", key, value}
}

func (s *Store) GetString(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.data[key]
	if !exists || entry.Type != TypeString || entry.IsExpired() {
		return "", false
	}
	return entry.StringVal, true
}

func (s *Store) ListPushLeft(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.data[key]
	if !exists || entry.Type != TypeList {
		entry = Entry{Type: TypeList, ListVal: []string{}}
	}
	entry.ListVal = append([]string{value}, entry.ListVal...)
	s.data[key] = entry
	s.aofChan <- []string{"LPUSH", key, value}
	s.replicaChan <- []string{"LPUSH", key, value}
}

func (s *Store) ListPopRight(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.data[key]
	if !exists || entry.Type != TypeList || len(entry.ListVal) == 0 || entry.IsExpired() {
		return "", false
	}
	value := entry.ListVal[len(entry.ListVal)-1]
	entry.ListVal = entry.ListVal[:len(entry.ListVal)-1]
	s.data[key] = entry
	return value, true
}

func (s *Store) SetAdd(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.data[key]
	if !exists || entry.Type != TypeSet {
		entry = Entry{Type: TypeSet, SetVal: make(map[string]struct{})}
	}
	entry.SetVal[value] = struct{}{}
	s.data[key] = entry
	s.aofChan <- []string{"SADD", key, value}
	s.replicaChan <- []string{"SADD", key, value}
}

func (s *Store) SetMembers(key string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.data[key]
	if !exists || entry.Type != TypeSet || entry.IsExpired() {
		return nil, false
	}
	members := make([]string, 0, len(entry.SetVal))
	for k := range entry.SetVal {
		members = append(members, k)
	}
	return members, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		s.aofChan <- []string{"DEL", key}
		s.replicaChan <- []string{"DEL", key}
		return true
	}
	return false
}

func (s *Store) ApplyReplicationCommand(args []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch args[0] {
	case "SET":
		s.data[args[1]] = Entry{Type: TypeString, StringVal: args[2]}
	case "LPUSH":
		entry, exists := s.data[args[1]]
		if !exists || entry.Type != TypeList {
			entry = Entry{Type: TypeList, ListVal: []string{}}
		}
		entry.ListVal = append([]string{args[2]}, entry.ListVal...)
		s.data[args[1]] = entry
	case "SADD":
		entry, exists := s.data[args[1]]
		if !exists || entry.Type != TypeSet {
			entry = Entry{Type: TypeSet, SetVal: make(map[string]struct{})}
		}
		entry.SetVal[args[2]] = struct{}{}
		s.data[args[1]] = entry
	case "DEL":
		delete(s.data, args[1])
	}
}

func (s *Store) GetSnapshot() map[string]Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]Entry, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	return snapshot
}

func (s *Store) LoadSnapshot(data map[string]Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = data
}

func (s *Store) ReplicaChan() chan []string {
	return s.replicaChan
}

func (s *Store) ExecuteTransaction(commands [][]string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]string, len(commands))
	for i, cmd := range commands {
		// Execute without propagating to AOF/replication here; do it post-transaction
		switch strings.ToUpper(cmd[0]) {
		case "SET":
			s.data[cmd[1]] = Entry{Type: TypeString, StringVal: cmd[2]}
			results[i] = "+OK\r\n"
		case "LPUSH":
			entry, exists := s.data[cmd[1]]
			if !exists || entry.Type != TypeList {
				entry = Entry{Type: TypeList, ListVal: []string{}}
			}
			entry.ListVal = append([]string{cmd[2]}, entry.ListVal...)
			s.data[cmd[1]] = entry
			results[i] = "+OK\r\n"
		case "SADD":
			entry, exists := s.data[cmd[1]]
			if !exists || entry.Type != TypeSet {
				entry = Entry{Type: TypeSet, SetVal: make(map[string]struct{})}
			}
			entry.SetVal[cmd[2]] = struct{}{}
			s.data[cmd[1]] = entry
			results[i] = "+OK\r\n"
		case "DEL":
			if _, exists := s.data[cmd[1]]; exists {
				delete(s.data, cmd[1])
				results[i] = ":1\r\n"
			} else {
				results[i] = ":0\r\n"
			}
		default:
			results[i] = fmt.Sprintf("-ERR unknown command '%s'\r\n", cmd[0])
		}
	}
	// Propagate to AOF and replication after successful execution
	for _, cmd := range commands {
		s.aofChan <- cmd
		s.replicaChan <- cmd
	}
	return results
}
