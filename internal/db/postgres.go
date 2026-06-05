package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB encapsulates the standard sql.DB pool to extend database-specific helper methods if needed.
type DB struct {
	*sql.DB
}

// ConnectPostgres initializes a connection pool to the primary PostgreSQL instance.
// It accepts connection parameters as arguments, which we will parse from environment variables later.
func ConnectPostgres(host, port, user, password, dbname string) (*DB, error) {
	// Construct the standard PostgreSQL connection string (DSN)
	// We explicitly target the 'pgx' driver name registered by the blank import.
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)

	dbPool, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection pool: %w", err)
	}

	// Production Connection Pool Lifecycles Configuration
	// This prevents resources from leaking or choking inside the Kubernetes cluster under heavy loads.
	dbPool.SetMaxOpenConns(25)
	dbPool.SetMaxIdleConns(25)                 // Keeps idle connections alive to avoid handshaking overhead
	dbPool.SetConnMaxLifetime(5 * time.Minute) // recycles stale connections safely

	// Establish a context timeout to verify the connection is genuinely alive
	if err := dbPool.Ping(); err != nil {
		dbPool.Close() // Safeguard resource leakage if ping fails
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &DB{dbPool}, nil
}
