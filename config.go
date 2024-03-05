package main

import "github.com/BurntSushi/toml"

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
}

func NewConfig() (*Config, error) {
	var cfg Config
	f := "config.toml"

	if _, err := toml.DecodeFile(f, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
