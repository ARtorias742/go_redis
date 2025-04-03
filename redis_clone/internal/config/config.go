package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port        string `yaml:"port"`
	LogLevel    string `yaml:"log_level"`
	RDBInterval int    `yaml:"rdb_interval"` // Seconds between RDB snapshots
	EnableAOF   bool   `yaml:"enable_aof"`
	ReplicaOf   string `yaml:"replica_of"` // Address of master if this is a replica
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Port == "" {
		cfg.Port = ":6379"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	return &cfg, nil
}
