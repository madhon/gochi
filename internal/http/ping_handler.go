package handler

import (
	"encoding/json"
	"github.com/go-chi/httplog/v2"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type pingHandler struct {
}

func NewPingHandler(r *chi.Mux) {
	handler := &pingHandler{}

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode("{ result: pong}")
	if err != nil {
		return
	}
}
