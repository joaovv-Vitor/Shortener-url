// Package config handles application configuration.
package config

import (
	"os"
)

// Config holds all application configuration.
type Config struct {
	// ServerPort is the port the HTTP server listens on.
	ServerPort string

	// BaseURL is the public-facing base URL used to construct short URLs.
	// Example: "http://localhost:8080"
	BaseURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
