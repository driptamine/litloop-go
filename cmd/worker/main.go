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
	c := config.Load()
	db, e := database.Open(c)
	if e != nil {
		log.Fatal(e)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if e = worker.New(db, c).Run(ctx); e != nil {
		log.Fatal(e)
	}
}
