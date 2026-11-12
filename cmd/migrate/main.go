package main

import (
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"log"
)

func main() {
	db, e := database.Open(config.Load())
	if e != nil {
		log.Fatal(e)
	}
	if e = database.Migrate(db); e != nil {
		log.Fatal(e)
	}
	log.Print("Schema migrated")
}
