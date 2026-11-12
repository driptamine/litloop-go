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
	c := config.Load()
	db, e := database.Open(c)
	if e != nil {
		slog.Error("database connection", "error", e)
		os.Exit(1)
	}
	if c.AutoMigrate {
		if e = database.Migrate(db); e != nil {
			slog.Error("migrate", "error", e)
			os.Exit(1)
		}
	}
	app := api.New(db, c)
	defer app.Hub.Close()
	s := &http.Server{Addr: c.Addr, Handler: app, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}()
	slog.Info("LitLoop listening", "addr", c.Addr)
	if e = s.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		slog.Error("serve", "error", e)
		os.Exit(1)
	}
}
