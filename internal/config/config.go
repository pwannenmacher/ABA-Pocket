package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// aus .env gelesene Werte (Fallback für nicht gesetzte Env-Variablen)
var dotenv map[string]string

func init() {
	dotenv, _ = godotenv.Read() // .env parsen, ohne os.Setenv aufzurufen
}

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	SessionSecret string
	AdminUsername string
	AdminPassword string
	DevMode       bool

	// Impressum
	ImprintName   string
	ImprintStreet string
	ImprintZip    string
	ImprintCity   string
	ImprintEmail  string
}

func Load() *Config {
	cfg := &Config{
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://aba:aba_password@localhost:5432/aba_pocket?sslmode=disable"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		AdminUsername: getEnv("ADMIN_USERNAME", ""),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		DevMode:       getEnv("DEV_MODE", "false") == "true",
		ImprintName:   getEnv("IMPRINT_NAME", ""),
		ImprintStreet: getEnv("IMPRINT_STREET", ""),
		ImprintZip:    getEnv("IMPRINT_ZIP", ""),
		ImprintCity:   getEnv("IMPRINT_CITY", ""),
		ImprintEmail:  getEnv("IMPRINT_EMAIL", ""),
	}

	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}
	if len(cfg.SessionSecret) < 32 {
		log.Fatal("SESSION_SECRET must be at least 32 characters")
	}

	return cfg
}

// getEnv prüft zuerst echte Env-Variablen (Docker), dann .env-Datei, dann Fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v, ok := dotenv[key]; ok && v != "" {
		return v
	}
	return fallback
}
