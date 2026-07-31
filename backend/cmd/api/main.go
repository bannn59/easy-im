package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/app"
	"easy-im/backend/internal/config"
	"easy-im/backend/internal/db"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		p, err := db.Open(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			log.Error("database open failed", "error", err)
			os.Exit(1)
		}
		pool = p
		defer pool.Close()
		log.Info("database connected", "service", "api")
	} else {
		log.Warn("DATABASE_URL unset; /readyz and auth will be unavailable", "service", "api")
	}

	if cfg.AuthJWTSecret == "" {
		log.Error("AUTH_JWT_SECRET unset; refusing to start. Set a production secret, or AUTH_DEV_INSECURE=1 for local development", "service", "api")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: app.NewAPIHandler(app.APIOptions{
			Pool:               pool,
			Log:                log,
			AuthJWTSecret:      cfg.AuthJWTSecret,
			AuthTokenTTL:       cfg.AuthTokenTTL,
			CORSAllowedOrigins: cfg.CORSAllowedOrigins,
			CookieSecure:       cfg.CookieSecure,
			CookieDomain:       cfg.CookieDomain,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api listening",
			"service", "api",
			"addr", cfg.Addr,
			"db_configured", pool != nil,
			"auth_configured", cfg.AuthJWTSecret != "",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen failed", "error", err)
			os.Exit(1)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
		os.Exit(1)
	}
	log.Info("api stopped", "service", "api")
}
