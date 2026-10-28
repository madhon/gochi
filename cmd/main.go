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
	"github.com/go-chi/httplog/v3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"

	_ "chier/docs"
	handler "chier/internal/http"
	"chier/internal/observability"

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
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With(
		slog.String("app", "chier"),
	)

	cfg, err := config.LoadAppConfig(".")
	if err != nil {
		logger.Error("unable to load configurations", ErrAttr(err))
		os.Exit(1)
	}

	traceInitCtx, cancelTraceInit := context.WithTimeout(context.Background(), 10*time.Second)
	shutdownTracing, err := observability.InitTracing(traceInitCtx)
	cancelTraceInit()
	if err != nil {
		logger.Error("unable to initialize tracing", ErrAttr(err))
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(middleware.Compress(5, "gzip"))
	router.Use(httplog.RequestLogger(logger, nil))
	router.Use(handler.TraceRoute)
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(middleware.RequestSize(1048576)) // 1MB limit
	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromXFFTrustedProxies(cfg.TrustedProxyCount))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/healthz"))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Adjust in production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false, // Disable if not needed
		MaxAge:           86400, // 24 hours
	}))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	limiter := rate.NewLimiter(
		rate.Every(12*time.Second),
		5)

	handler.NewPingHandler(router, limiter)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := newServer(cfg.ServeAddress, otelhttp.NewHandler(router, "chier"))

	logger.Info("Server started....", slog.String("address", cfg.ServeAddress))

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	serverStopErr := waitForShutdown(ctx, server, serverErr, logger)

	traceShutdownCtx, cancelTraceShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	if err := shutdownTracing(traceShutdownCtx); err != nil {
		logger.Error("tracing shutdown failed", ErrAttr(err))
	}
	cancelTraceShutdown()

	if serverStopErr != nil {
		logger.Error("server stopped with error", ErrAttr(serverStopErr))
		stop()
		os.Exit(1)
	}

	logger.Info("Server stopped gracefully")
}

func waitForShutdown(ctx context.Context, server *http.Server, serverErr <-chan error, logger *slog.Logger) error {
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("Shutdown signal received, gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", ErrAttr(err))
		return errors.Join(err, server.Close())
	}
	return nil
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
