package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	dapr "github.com/dapr/go-sdk/client"
)

const defaultStoreName = "statestore"

// StateClient is the subset of Dapr client used for state operations.
type StateClient interface {
	GetState(ctx context.Context, storeName, key string, meta map[string]string) (*dapr.StateItem, error)
	SaveState(ctx context.Context, storeName, key string, data []byte, meta map[string]string, opts ...dapr.StateOption) error
	DeleteState(ctx context.Context, storeName, key string, meta map[string]string) error
}

// Server is an HTTP server that uses Dapr for state management.
type Server struct {
	client    StateClient
	storeName string
	log       *slog.Logger
}

// New creates a new Server.
func New(c StateClient, storeName string, log *slog.Logger) *Server {
	if storeName == "" {
		storeName = defaultStoreName
	}
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		client:    c,
		storeName: storeName,
		log:       log,
	}
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /state/{key}", s.getState)
	mux.HandleFunc("POST /state/{key}", s.saveState)
	mux.HandleFunc("DELETE /state/{key}", s.deleteState)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/state/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	item, err := s.client.GetState(r.Context(), s.storeName, key, nil)
	if err != nil {
		s.log.Error("get state failed", "key", key, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if item == nil || len(item.Value) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(item.Value)
}

func (s *Server) saveState(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/state/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	if err := s.client.SaveState(r.Context(), s.storeName, key, data, nil); err != nil {
		s.log.Error("save state failed", "key", key, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteState(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/state/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if err := s.client.DeleteState(r.Context(), s.storeName, key, nil); err != nil {
		s.log.Error("delete state failed", "key", key, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
