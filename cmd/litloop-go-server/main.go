package main

import (
	"context"
	api "litloop/backend/app"
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	appConfig := config.Load()
	db, err := database.Open(appConfig)
	if err != nil {
		slog.Error("database connection", "error", err)
		os.Exit(1)
	}
	if appConfig.AutoMigrate {
		if err = database.Migrate(db); err != nil {
			slog.Error("migrate", "error", err)
			os.Exit(1)
		}
	}
	app := api.New(db, appConfig)
	defer app.Hub.Close()
	server := &http.Server{Addr: appConfig.Addr, Handler: app, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("LitLoop listening", "addr", appConfig.Addr)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("serve", "error", err)
		os.Exit(1)
	}
}
