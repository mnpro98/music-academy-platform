package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	*sql.DB
}

func ConnectPostgres(host, port, user, password, dbname string) (*DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)

	dbPool, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection pool: %w", err)
	}

	dbPool.SetMaxOpenConns(25)
	dbPool.SetMaxIdleConns(25)
	dbPool.SetConnMaxLifetime(5 * time.Minute)

	if err := dbPool.Ping(); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &DB{dbPool}, nil
}
