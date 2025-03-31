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
	}

	router := chi.NewRouter()

	router.Use(httplog.RequestLogger(logger))
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/healthz"))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	limiter := rate.NewLimiter(rate.Every(12*time.Second), 5)
	handler.NewPingHandler(router, limiter)

	server := newServer(cfg.ServeAddress, router)

	logger.Info("Server started....", slog.String("address", cfg.ServeAddress))

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", ErrAttr(err))
		}
	}()

	waitForShutdown(server)
}

func waitForShutdown(server *http.Server) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		logger.Error("Error occurred during shutdown", ErrAttr(err))
		return
	}
}

func newServer(addr string, r *chi.Mux) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
