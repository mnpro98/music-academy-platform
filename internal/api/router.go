package api

import (
	"net/http"
	"time"

	"music-academy-platform/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Router *chi.Mux
	DB     *db.DB
}

func NewServer(database *db.DB) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	s := &Server{
		Router: r,
		DB:     database,
	}

	s.routes()

	return s
}

func (s *Server) routes() {
	s.Router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "UP"}`))
	})

	s.Router.Route("/api/v1", func(r chi.Router) {
		r.Post("/students", s.HandleRegisterStudent())
	})
}
