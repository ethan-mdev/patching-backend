package config

import (
	"fmt"
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

func Load() (*Config, error) {
	// Load .env file
	godotenv.Load()

	return &Config{
		FilesDir:    os.Getenv("FILES_DIR"),
		Port:        os.Getenv("PORT"),
		JWKSUrl:     os.Getenv("JWKS_URL"),
		JWKSRefresh: getDurationEnv("JWKS_REFRESH_MINUTES", 5) * time.Minute,
	}, nil
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
