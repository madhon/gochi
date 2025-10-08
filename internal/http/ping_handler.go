package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"golang.org/x/time/rate"
)

type pingResponse struct {
	Result string `json:"result"`
}

type pingHandler struct {
	limiter      *rate.Limiter
	responsePool sync.Pool
}

func NewPingHandler(r *chi.Mux, l *rate.Limiter) {
	handler := &pingHandler{
		limiter: l,
		responsePool: sync.Pool{
			New: func() interface{} {
				return &pingResponse{}
			},
		},
	}

	r.Route("/v1", func(r chi.Router) {
		r.Get("/ping", handler.GetPing)
	})
}

// GetPing godoc
// @Summary  Ping the API
// @Description Pings the API and gets response back
// @Produce json
// @Router       /v1/ping [get]
// @Success 200 {object} pingResponse
// @Failure 429 {string} string "Rate limit exceeded"
func (h *pingHandler) GetPing(w http.ResponseWriter, r *http.Request) {
	oplog := slog.Default()
	oplog.Info("Ping Handler Called")

	if h.limiter != nil && !h.limiter.Allow() {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := h.responsePool.Get().(*pingResponse)
	response.Result = "pong"
	defer h.responsePool.Put(response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
