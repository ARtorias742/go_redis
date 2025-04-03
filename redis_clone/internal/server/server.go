package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/ARtorias742/Redis/internal/commands"
	"github.com/ARtorias742/Redis/internal/config"
	"github.com/ARtorias742/Redis/internal/store"
	"github.com/ARtorias742/Redis/internal/types"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg       *config.Config
	store     *store.Store
	ln        net.Listener
	log       *logrus.Logger
	clients   map[net.Conn]*types.ClientState
	clientsMu sync.Mutex
}

func NewServer(cfg *config.Config) *Server {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{})
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	log.SetLevel(level)
	file, _ := os.OpenFile("logs/redis.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(file)

	s := &Server{
		cfg:     cfg,
		store:   store.NewStore(),
		log:     log,
		clients: make(map[net.Conn]*types.ClientState),
	}

	if cfg.ReplicaOf != "" {
		go s.startReplication(cfg.ReplicaOf)
	}
	return s
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.Port)
	if err != nil {
		return err
	}
	s.ln = ln
	s.log.Infof("Server listening on %s", s.cfg.Port)

	go s.store.StartPersistence(s.cfg.RDBInterval, s.cfg.EnableAOF)

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.log.Errorf("Accept error: %v", err)
			continue
		}
		s.clientsMu.Lock()
		s.clients[conn] = &types.ClientState{}
		s.clientsMu.Unlock()
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		conn.Close()
	}()
	reader := bufio.NewReader(conn)
	s.log.Infof("New connection from %s", conn.RemoteAddr().String())

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			s.log.Infof("Connection closed: %s", conn.RemoteAddr().String())
			return
		}
		line = strings.TrimSpace(line)
		args := strings.Split(line, " ")
		if len(args) == 0 {
			continue
		}

		s.clientsMu.Lock()
		clientState := s.clients[conn]
		s.clientsMu.Unlock()

		var response string
		if strings.ToUpper(args[0]) == "SYNC" {
			s.log.Infof("Handling SYNC request from %s", conn.RemoteAddr().String())
			snapshot := s.store.GetSnapshot()
			data, _ := json.Marshal(snapshot)
			fmt.Fprintf(conn, "%s\n", string(data))
			for cmd := range s.store.ReplicaChan() {
				fmt.Fprintf(conn, "%s\n", strings.Join(cmd, " "))
			}

			return // SYNC connection becomes a streaming connection

		} else if clientState.InTransaction && args[0] != "EXEC" && args[0] != "DISCARD" {
			clientState.QueuedCommands = append(clientState.QueuedCommands, args)
			response = "+QUEUED\r\n"
		} else {
			response = commands.ExecuteCommand(s.store, args, clientState)
		}

		if _, err := conn.Write([]byte(response + "\n")); err != nil {
			s.log.Errorf("Write error: %v", err)
			return
		}
	}
}
