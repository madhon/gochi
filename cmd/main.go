package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chier/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"golang.org/x/time/rate"

	_ "chier/docs"
	handler "chier/internal/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample chi server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @Host localhost:4343
// @BasePath /v1

var logger httplog.Logger

func main() {
	logger := httplog.NewLogger("Chi", httplog.Options{
		Concise:          true,
		RequestHeaders:   true,
		TimeFieldFormat:  time.RFC3339,
		MessageFieldName: "message",
	})

	cfg, err := config.LoadAppConfig(".")
	if err != nil {
		logger.Error("unable to load configurations", ErrAttr(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(httplog.RequestLogger(logger))
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/healthz"))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Adjust in production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	limiter := rate.NewLimiter(
		rate.Every(12*time.Second),
		5)

	handler.NewPingHandler(router, limiter)

	server := newServer(cfg.ServeAddress, router)

	logger.Info("Server started....", slog.String("address", cfg.ServeAddress))

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", ErrAttr(err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	waitForShutdown(server)
	logger.Info("Server stopped gracefully")
}

func waitForShutdown(server *http.Server) {
	// Create channel for shutdown signals
	sig := make(chan os.Signal, 1)
	// Listen for interrupt and SIGTERM signals
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	logger.Info("Shutdown signal received, gracefully shutting down...")

	// Create timeout context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", ErrAttr(err))
		// Force shutdown
		if err := server.Close(); err != nil {
			logger.Error("Server forced close failed", ErrAttr(err))
		}
	}
}
func newServer(addr string, r *chi.Mux) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
