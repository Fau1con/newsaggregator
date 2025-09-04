package main

import (
	"log"
	"news/internal/app"
	"news/internal/config"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("FATAL: could not load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("FATAL: invalid config: %v", err)
	}
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("FATAL: could not create app: %v", err)
	}
	if err := application.Run(); err != nil {
		log.Fatalf("FATAL: app failed: %v", err)
	}
}
