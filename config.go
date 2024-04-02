package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ResetES       bool
	ESAddresses   []string
	IndexName     string
	MecabDir      string
	BulkSourceDir string
	BulkWorkerNum int
	BulkESUnitNum int
	IsBulkSubdir  bool
	AbortOnError  bool
	CacheSize     int64
}

func NewConfig() (*Config, error) {
	var cfg Config
	f := "config.toml"

	if _, err := toml.DecodeFile(f, &cfg); err != nil {
		return nil, err
	}

	f = "config.toml.local"
	if _, err := os.Stat(f); err == nil {
		if _, err := toml.DecodeFile(f, &cfg); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
