package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/ARtorias742/redis_clone/internal/commands"
	"github.com/ARtorias742/redis_clone/internal/store"
)

type Server struct {
	addr  string
	store *store.Store
}

func NewServer(addr string) *Server {
	return &Server{
		addr:  addr,
		store: store.NewStore(),
	}
}

func (s *Server) Start() error {

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Println("Server listening on %s\n", s.addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue

		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		args := strings.Split(line, " ")
		if len(args) == 0 {
			continue
		}

		response := commands.ExecuteCommand(s.store, args)
		_, err = conn.Write([]byte(response + "\n"))
		if err != nil {
			return
		}
	}
}
