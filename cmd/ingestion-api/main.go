package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"music-academy-platform/internal/api"
	"music-academy-platform/internal/config"
	"music-academy-platform/internal/db"
	"music-academy-platform/internal/worker"
)

func main() {
	log.Println("Initializing Music Academy Ingestion API & Outbox Publisher Service...")

	cfg := config.Load()

	// Initialize Primary Database Connection Pool (PostgreSQL)
	log.Printf("Connecting to primary database at %s:%s...\n", cfg.DBHost, cfg.DBPort)
	database, err := db.ConnectPostgres(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Critical Error: Database initialization failed: %v", err)
	}
	defer func() {
		log.Println("Closing database connection pool...")
		if err := database.Close(); err != nil {
			log.Printf("Error closing database connection pool: %v\n", err)
		}
	}()

	// Initialize and Start the Outbox Background Publisher
	log.Printf("Connecting Outbox Worker to NATS Broker at %s...\n", cfg.NATSURL)
	outboxPub, err := worker.NewOutboxPublisher(database, cfg.NATSURL, cfg.OutboxInterval)
	if err != nil {
		log.Fatalf("Critical Error: Failed to initialize Outbox Publisher: %v", err)
	}
	defer outboxPub.Close()

	// Create a cancelable context to control the background worker's lifecycle
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	// Launch the Outbox Publisher loop inside a background Goroutine
	go func() {
		outboxPub.Start(workerCtx)
	}()

	// Initialize Go-Chi Router Engine and Inject Dependencies
	serverApp := api.NewServer(database)

	// Configure the Underlying HTTP Server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      serverApp.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to catch lifecycle interruption OS signals
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	// Spin up the HTTP server engine inside a background Goroutine
	go func() {
		log.Printf("Ingestion API Service listening on port %s\n", cfg.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Critical Error: Server forced to shut down: %v", err)
		}
	}()

	// --- BLOCKING LIFECYCLE MONITOR ---
	sig := <-shutdownSignal
	log.Printf("Termination signal captured (%s). Commencing graceful shutdown protocol...\n", sig)

	// Stop the background outbox worker loop first
	log.Println("Signaling background outbox worker to halt...")
	cancelWorker()

	// Enforce a safety timeout ceiling for active HTTP transactions to wrap up
	shutdownCtx, cancelServer := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelServer()

	// Instruct the server to stop accepting new traffic and gracefully drain open connections
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server aggressive shutdown forced: %v", err)
	}

	log.Println("Ingestion API Service gracefully stopped. All resources discharged.")
}
