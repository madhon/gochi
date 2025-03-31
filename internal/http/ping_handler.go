package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
	"golang.org/x/time/rate"

	"github.com/go-chi/chi/v5"
)

type pingHandler struct {
	l *rate.Limiter
}

func NewPingHandler(r *chi.Mux, l *rate.Limiter) {
	handler := &pingHandler{}
	handler.l = l

	r.Route("/v1/ping", func(r chi.Router) {
		r.Get("/", handler.GetPing)
	})
}

// GetPing godoc
// @Summary  Ping the API
// @Description Pings the API and gets response back
// @Produce json
// @Router       /ping [get]
func (h *pingHandler) GetPing(w http.ResponseWriter, r *http.Request) {
	oplog := httplog.LogEntry(r.Context())
	oplog.Info("Ping Handler Called")

	if h.l != nil {
		if !h.l.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode("{ result: pong}")
	if err != nil {
		return
	}
}
