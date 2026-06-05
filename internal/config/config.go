package config

import (
	"os"
	"time"
)

// Config aggregates all operational environment parameters required
// to boot and connect the microservices.
type Config struct {
	ServerPort string

	// Primary Database Configuration (PostgreSQL)
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Message Broker Configuration (NATS JetStream)
	NATSURL string

	OutboxInterval time.Duration // Added for high-frequency tuning
}

// Load reads values from the system environment variables.
// If a variable is missing, it injects a sane local development default.
func Load() *Config {
	intervalStr := getEnv("OUTBOX_INTERVAL", "200ms")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		interval = 200 * time.Millisecond
	}

	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "music_academy"),

		NATSURL: getEnv("NATS_URL", "nats://localhost:4222"),

		OutboxInterval: interval,
	}
}

// getEnv is a private helper function that fetches an environment variable
// or falls back to a provided default string if the variable is uninitialized.
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
