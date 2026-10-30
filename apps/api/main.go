package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/handlers"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	providerName := strings.ToLower(env("CODENET_PROVIDER", "mock"))
	var provider providers.RouterProvider
	if providerName == "linux" {
		provider = providers.NewLinux()
	} else {
		provider = providers.NewMock()
	}
	address := env("CODENET_API_ADDRESS", "127.0.0.1:8080")
	allowedOrigin := env("CODENET_ALLOWED_ORIGIN", "http://127.0.0.1:5173")
	server := &http.Server{Addr: address, Handler: handlers.New(services.NewRouter(provider), logger, allowedOrigin), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		logger.Info("CODENET Management API berjalan", "address", address, "provider", providerName)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server berhenti", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown gagal", "error", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
