// Package config handles application configuration.
package config

import (
	"os"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	// ServerPort is the port the HTTP server listens on.
	ServerPort string

	// BaseURL is the public-facing base URL used to construct short URLs.
	// Example: "http://localhost:8080"
	BaseURL string

	// HashSalt is the secret salt used by Hashids to generate
	// non-sequential, obfuscated short codes.
	HashSalt string

	// CassandraHosts is a comma-separated list of Cassandra contact points.
	// Example: "127.0.0.1" or "cassandra-1,cassandra-2"
	CassandraHosts []string

	// CassandraKeyspace is the Cassandra keyspace to use.
	CassandraKeyspace string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	hostsRaw := getEnv("CASSANDRA_HOSTS", "127.0.0.1")
	hosts := strings.Split(hostsRaw, ",")
	for i := range hosts {
		hosts[i] = strings.TrimSpace(hosts[i])
	}

	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
		HashSalt:          getEnv("HASH_SALT", "shorner-default-dev-salt"),
		CassandraHosts:    hosts,
		CassandraKeyspace: getEnv("CASSANDRA_KEYSPACE", "shorner"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
