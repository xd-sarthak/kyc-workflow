package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL    string
	JWTSecret      string
	StorageBackend string // "local" or "s3"
	StorageRoot    string
	AWSBucket      string
	AWSRegion      string
	Port           int
}

// Load reads configuration from environment variables and validates required fields.
// It automatically loads a .env file if one exists, without overriding already-set env vars.
func Load() (*Config, error) {
	// Load .env file if present. Ignore error (file may not exist in production).
	_ = godotenv.Load()
	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		StorageBackend: getEnvOrDefault("STORAGE_BACKEND", "local"),
		StorageRoot:    getEnvOrDefault("STORAGE_ROOT", "./uploads"),
		AWSBucket:      os.Getenv("AWS_BUCKET"),
		AWSRegion:      os.Getenv("AWS_REGION"),
	}

	portStr := getEnvOrDefault("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT value %q: %w", portStr, err)
	}
	cfg.Port = port

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if cfg.StorageBackend == "s3" {
		if cfg.AWSBucket == "" {
			return nil, fmt.Errorf("AWS_BUCKET is required when STORAGE_BACKEND=s3")
		}
		if cfg.AWSRegion == "" {
			return nil, fmt.Errorf("AWS_REGION is required when STORAGE_BACKEND=s3")
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
