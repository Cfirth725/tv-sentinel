package database

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port         string `json:"PORT"`
	DatabasePath string `json:"DATABASE_PATH"`
	WALMode      bool   `json:"SQLITE_WAL_MODE"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
