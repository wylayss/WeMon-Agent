package main

import (
	"flag"
	"log"
	"os"

	"wemon-agent/api"
	"wemon-agent/collector"
	"wemon-agent/config"
)

func main() {
	configPath := flag.String("config", "/etc/wemon-agent/config.json", "Path to WeMon-Agent configuration file")
	flag.Parse()

	// If default configuration path doesn't exist, fallback to local config.json for dev environments
	if _, err := os.Stat(*configPath); os.IsNotExist(err) && *configPath == "/etc/wemon-agent/config.json" {
		if _, localErr := os.Stat("config.json"); localErr == nil {
			*configPath = "config.json"
			log.Println("Using local config.json for development")
		}
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	tokenPrefix := "none"
	if len(cfg.NodeToken) >= 6 {
		tokenPrefix = cfg.NodeToken[:6]
	} else if len(cfg.NodeToken) > 0 {
		tokenPrefix = cfg.NodeToken
	}

	log.Printf("WeMon Agent starting (Server: %s, Token prefix: %s)...", cfg.ServerURL, tokenPrefix)

	col := collector.NewCollector()
	client := api.NewClient(cfg, col)

	client.Start()
}
