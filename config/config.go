package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ServerURL          string `json:"server_url"`
	NodeToken          string `json:"node_token"`
	IntervalSeconds    int    `json:"interval_seconds"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

// LoadConfig reads the configuration file from the specified path.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 5 // Default to 5 seconds
	}

	return &cfg, nil
}
