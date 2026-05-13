package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joaovv-Vitor/Shorner-url/internal/config"
	"github.com/joaovv-Vitor/Shorner-url/internal/shorner"
)

func main() {
	// Setup structured logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration.
	cfg := config.Load()

	// Wire dependencies (in-memory adapters for now).
	repo := shorner.NewMemoryRepository()
	idGenerator := shorner.NewMemoryIDGenerator()
	urlService := shorner.NewService(repo, idGenerator)
	httpHandler := shorner.NewHandler(urlService, cfg.BaseURL, logger)

	// Setup HTTP server.
	mux := http.NewServeMux()
	httpHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		logger.Info("server starting",
			slog.String("port", cfg.ServerPort),
			slog.String("base_url", cfg.BaseURL),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
