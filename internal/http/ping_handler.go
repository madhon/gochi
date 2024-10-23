package handler

import (
	"encoding/json"
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

func (h *pingHandler) GetPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("{ result: pong}")

}
