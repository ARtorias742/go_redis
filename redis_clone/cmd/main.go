package main

import (
	"fmt"

	"github.com/ARtorias742/redis_clone/internal/server"
)

func main() {
	fmt.Println("Starting Redis Clone...")
	s := server.NewServer(":6379")
	if err := s.Start(); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
