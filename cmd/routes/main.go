package main

import (
	"encoding/json"
	api "litloop/backend/app"
	"litloop/backend/internal/config"
	"os"
)

func main() {
	server := api.New(nil, config.Config{})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(server.OpenAPI()); err != nil {
		panic(err)
	}
}
