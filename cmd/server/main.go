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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gocql/gocql"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"

	"github.com/joaovv-Vitor/Shorner-url/internal/config"
	"github.com/joaovv-Vitor/Shorner-url/internal/shorner"
	hashids "github.com/joaovv-Vitor/Shorner-url/pkg/hashids_generator"
)

func main() {
	// Setup structured logger with colored output.
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05",
	}))

	// Load .env file if it exists (dev only, ignored in production).
	_ = godotenv.Load()

	// Load configuration.
	cfg := config.Load()

	// Initialize short code encoder (Hashids + Base62 + salt).
	encoder, err := hashids.NewGenerator(cfg.HashSalt, 4, 7)
	if err != nil {
		logger.Error("failed to initialize shortcode encoder", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to Redis.
	ctx := context.Background()
	redisClient, err := shorner.NewRedisClient(ctx, shorner.RedisConfig{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	logger.Info("connected to redis", slog.String("addr", cfg.RedisAddr))

	// Connect to Cassandra.
	cassandraSession, err := shorner.NewCassandraSession(shorner.CassandraConfig{
		Hosts:       cfg.CassandraHosts,
		Keyspace:    cfg.CassandraKeyspace,
		Consistency: gocql.LocalOne,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		logger.Error("failed to connect to cassandra", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer cassandraSession.Close()

	logger.Info("connected to cassandra",
		slog.Any("hosts", cfg.CassandraHosts),
		slog.String("keyspace", cfg.CassandraKeyspace),
	)

	// Wire dependencies.
	cassandraRepo := shorner.NewCassandraRepository(cassandraSession)
	repo := shorner.NewCachedRepository(cassandraRepo, redisClient, 24*time.Hour) // Cache TTL: 24h
	idGenerator := shorner.NewRedisIDGenerator(redisClient, "shorner:id_counter")
	urlService := shorner.NewService(repo, idGenerator, encoder)
	httpHandler := shorner.NewHandler(urlService, cfg.BaseURL, logger)

	// Setup HTTP server with chi router.
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(shorner.RequestLogger(logger))
	httpHandler.RegisterRoutes(r)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      r,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	cassandraSession.Close()
	redisClient.Close()
	logger.Info("server stopped gracefully")
}
