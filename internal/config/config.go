package config

import (
	"encoding/json"
	"os"
)

type QueueConfig struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	MaxSub int    `json:"max_sub"`
}

type Config struct {
	Queues []QueueConfig `json:"queues"`
	Addr   string        `json:"addr"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
