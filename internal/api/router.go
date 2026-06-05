package api

import (
	"net/http"
	"time"

	"music-academy-platform/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server encapsulates the application dependencies (like the database pool)
// and exposes them securely to the HTTP handler methods.
type Server struct {
	Router *chi.Mux
	DB     *db.DB
}

// NewServer initializes a new Go-Chi router, attaches global middleware,
// maps API endpoints, and returns a fully configured Server instance.
func NewServer(database *db.DB) *Server {
	r := chi.NewRouter()

	// Standard production middlewares
	r.Use(middleware.RequestID)                 // Injects a unique ID into each request context for distributed tracing
	r.Use(middleware.RealIP)                    // Captures the actual client IP address when running behind a K8s ingress proxy
	r.Use(middleware.Logger)                    // Cleanly logs incoming requests, status codes, and latency to stdout
	r.Use(middleware.Recoverer)                 // Gracefully recovers from code panics, preventing the whole API container
	r.Use(middleware.Timeout(60 * time.Second)) // Enforces an absolute safety ceiling for sluggish client requests

	s := &Server{
		Router: r,
		DB:     database,
	}

	// Register API routes
	s.routes()

	return s
}

// routes handles the clean mapping of HTTP endpoints to their respective logic blocks.
func (s *Server) routes() {
	// Base healthcheck route (critical for Kubernetes liveness and readiness probes)
	s.Router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "UP"}`))
	})

	// API version grouping
	s.Router.Route("/api/v1", func(r chi.Router) {
		// Ingestion endpoint for student registrations
		r.Post("/students", s.HandleRegisterStudent())
	})
}
