package config

import (
	"os"
	"time"
)

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	NATSURL        string
	OutboxInterval time.Duration
}

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

		NATSURL:        getEnv("NATS_URL", "nats://localhost:4222"),
		OutboxInterval: interval,
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
