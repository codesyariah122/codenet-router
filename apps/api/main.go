package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/auth"
	"github.com/codesyariah/codenet-router/apps/api/internal/handlers"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databasePath := env("CODENET_DB_PATH", "data/codenet.db")
	store, err := auth.OpenStore(databasePath)
	if err != nil {
		logger.Error("database autentikasi tidak dapat dibuka", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	adminCount, err := store.AdminCount(context.Background())
	if err != nil {
		logger.Error("status administrator tidak dapat dibaca", "error", err)
		os.Exit(1)
	}
	if adminCount == 0 {
		logger.Error("administrator belum diinisialisasi; jalankan codenet-admin init sebelum memulai API")
		os.Exit(1)
	}
	authService := auth.NewService(store, auth.Config{SessionTTL: envDuration("CODENET_SESSION_TTL", 8*time.Hour), LoginMaxAttempts: envInt("CODENET_LOGIN_MAX_ATTEMPTS", 5), LoginWindow: envDuration("CODENET_LOGIN_WINDOW", 15*time.Minute)})
	providerName := strings.ToLower(env("CODENET_PROVIDER", "mock"))
	var provider providers.RouterProvider
	if providerName == "linux" {
		provider = providers.NewLinux()
	} else {
		provider = providers.NewMock()
	}
	address := env("CODENET_API_ADDRESS", "127.0.0.1:8080")
	allowedOrigin := env("CODENET_ALLOWED_ORIGIN", "http://127.0.0.1:5173")
	server := &http.Server{Addr: address, Handler: handlers.New(services.NewRouter(provider), authService, logger, handlers.Config{AllowedOrigin: allowedOrigin, CookieName: "codenet_session", CookieSecure: envBool("CODENET_COOKIE_SECURE", false)}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
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

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
