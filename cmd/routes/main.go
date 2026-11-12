package main

import (
	"encoding/json"
	api "litloop/backend/app"
	"litloop/backend/internal/config"
	"os"
)

func main() {
	s := api.New(nil, config.Config{})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(s.OpenAPI()); e != nil {
		panic(e)
	}
}
