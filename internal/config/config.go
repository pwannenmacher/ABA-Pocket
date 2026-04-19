package config

import (
	"log"
	"os"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	SessionSecret string
	AdminUsername string
	AdminPassword string
	DevMode       bool
}

func Load() *Config {
	cfg := &Config{
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://aba:aba_password@localhost:5432/aba_pocket?sslmode=disable"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		AdminUsername: getEnv("ADMIN_USERNAME", ""),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		DevMode:       getEnv("DEV_MODE", "false") == "true",
	}

	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}
	if len(cfg.SessionSecret) < 32 {
		log.Fatal("SESSION_SECRET must be at least 32 characters")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
