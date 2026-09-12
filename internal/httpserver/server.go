// Package httpserver provides the CareerPilot HTTP API server.
package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Server holds the HTTP handler dependencies for CareerPilot's API.
type Server struct {
	logger *slog.Logger
	mux    *http.ServeMux
}

// New builds an HTTP server with routes registered.
func New(logger *slog.Logger) *Server {
	s := &Server{
		logger: logger,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

type healthResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to encode health response", "error", err)
	}
}
