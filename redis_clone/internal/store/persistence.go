package store

import (
	"encoding/json"
	"os"
	"time"
)

func (s *Store) StartPersistence(rdbInterval int, enableAOF bool) {
	go func() {
		ticker := time.NewTicker(time.Duration(rdbInterval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.saveRDB()
		}
	}()

	if enableAOF {
		go func() {
			file, _ := os.OpenFile("data/appendonly.aof", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			defer file.Close()
			for cmd := range s.aofChan {
				jsonCmd, _ := json.Marshal(cmd)
				file.WriteString(string(jsonCmd) + "\n")
			}
		}()
	}
}

func (s *Store) saveRDB() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.data)
	os.WriteFile("data/dump.rdb", data, 0644)
}

func (s *Store) LoadRDB() error {
	data, err := os.ReadFile("data/dump.rdb")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.data)
}
