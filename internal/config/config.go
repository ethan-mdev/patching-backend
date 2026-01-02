package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	FilesDir    string
	Port        string
	JWKSUrl     string
	JWKSRefresh time.Duration
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found")
	}

	return &Config{
		FilesDir:    os.Getenv("FILES_DIR"),
		Port:        os.Getenv("PORT"),
		JWKSUrl:     os.Getenv("JWKS_URL"),
		JWKSRefresh: getDurationEnv("JWKS_REFRESH_MINUTES", 5) * time.Minute,
	}
}

func getDurationEnv(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		var duration int
		if _, err := fmt.Sscanf(value, "%d", &duration); err == nil {
			return time.Duration(duration)
		}
	}
	return time.Duration(defaultValue)
}
