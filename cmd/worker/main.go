package main

import (
	"context"
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"litloop/backend/internal/worker"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	appConfig := config.Load()
	db, err := database.Open(appConfig)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err = worker.New(db, appConfig).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
