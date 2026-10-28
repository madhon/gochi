package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TraceRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		routeCtx := chi.RouteContext(r.Context())
		if routeCtx == nil {
			return
		}

		route := routeCtx.RoutePattern()
		if route == "" {
			return
		}

		span := trace.SpanFromContext(r.Context())
		if span.IsRecording() {
			span.SetName(r.Method + " " + route)
			span.SetAttributes(attribute.String("http.route", route))
		}
	})
}
