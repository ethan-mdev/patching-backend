package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	FilesDir       string
	Port           string
	JWKSUrl        string
	JWKSRefresh    time.Duration
	AllowedOrigins string
	Environment    string
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		FilesDir:       getEnv("FILES_DIR", "./files"),
		Port:           getEnv("PORT", "8081"),
		JWKSUrl:        getEnv("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
		JWKSRefresh:    getDurationEnv("JWKS_REFRESH_MINUTES", 15) * time.Minute,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		Environment:    getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
