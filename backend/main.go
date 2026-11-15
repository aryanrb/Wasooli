package main

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"wasooli/config"
	"wasooli/internal/db"
	"wasooli/internal/db/sqlc"
	"wasooli/internal/handlers"
	"wasooli/internal/server"
	"wasooli/internal/services"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	log, _ := zap.NewProduction()
	defer func() { _ = log.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config", zap.Error(err))
	}

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, migrationsFS); err != nil {
		log.Fatal("migrate", zap.Error(err))
	}

	q := sqlc.New(pool)
	svc := services.New(q, log)
	h := handlers.New(svc, cfg, log)
	e := server.New(cfg, log, h)

	if cfg.GoogleClientID == "" {
		log.Warn("GOOGLE_CLIENT_ID not set — Google sign-in is disabled")
	}

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server", zap.Error(err))
		}
	}()
	log.Info("service started", zap.String("port", cfg.Port), zap.String("env", cfg.Env))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", zap.Error(err))
	}
	log.Info("service stopped")
}
