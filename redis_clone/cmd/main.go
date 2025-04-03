package main

import (
	"flag"
	"fmt"

	"github.com/ARtorias742/Redis/internal/config"
	"github.com/ARtorias742/Redis/internal/server"
)

func main() {

	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	fmt.Println("Starting Redis Clone...")
	s := server.NewServer(cfg)
	if err := s.Start(); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
