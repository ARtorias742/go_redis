package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ARtorias742/Redis/internal/store"
)

func (s *Server) startReplication(masterAddr string) {
	for {
		conn, err := net.Dial("tcp", masterAddr)
		if err != nil {
			s.log.Errorf("Failed to connect to master %s: %v", masterAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		s.log.Infof("Connected to master at %s", masterAddr)
		reader := bufio.NewReader(conn)

		// Request full sync
		fmt.Fprintf(conn, "SYNC\n")
		snapshot, err := reader.ReadString('\n')
		if err != nil {
			s.log.Errorf("SYNC error: %v", err)
			conn.Close()
			continue
		}
		var data map[string]store.Entry
		if err := json.Unmarshal([]byte(snapshot), &data); err != nil {
			s.log.Errorf("Failed to decode snapshot: %v", err)
			conn.Close()
			continue
		}
		s.store.LoadSnapshot(data)

		// Stream ongoing commands
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				s.log.Errorf("Replication error: %v", err)
				break
			}
			args := strings.Split(strings.TrimSpace(line), " ")
			s.store.ApplyReplicationCommand(args)
		}
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

func (s *Server) handleSync(conn net.Conn) {
	s.log.Infof("Handling SYNC request from %s", conn.RemoteAddr().String())
	snapshot := s.store.GetSnapshot()
	data, _ := json.Marshal(snapshot)
	fmt.Fprintf(conn, "%s\n", string(data))

	// Stream subsequent commands (simplified)
	for cmd := range s.store.ReplicaChan() {
		fmt.Fprintf(conn, "%s\n", strings.Join(cmd, " "))
	}
}
