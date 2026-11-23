package main

import (
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"log"
)

func main() {
	db, err := database.Open(config.Load())
	if err != nil {
		log.Fatal(err)
	}
	if err = database.Migrate(db); err != nil {
		log.Fatal(err)
	}
	log.Print("Schema migrated")
}
