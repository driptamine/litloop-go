package httpapi

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

var routeParam = regexp.MustCompile(`\{([^}:]+)(?::[^}]+)?\}`)

func (server *Server) OpenAPI() any {
	paths := map[string]any{}
	_ = chi.Walk(server.Router, func(method, path string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(path, "/ws/") {
			return nil
		}
		Clean := routeParam.ReplaceAllString(path, "{$1}")
		if strings.HasSuffix(Clean, "/*") {
			Clean = strings.TrimSuffix(Clean, "*") + "{path}"
		}
		item, _ := paths[Clean].(map[string]any)
		if item == nil {
			item = map[string]any{}
			paths[Clean] = item
		}
		parameters := []any{}
		for _, match := range routeParam.FindAllStringSubmatch(Clean, -1) {
			parameters = append(parameters, map[string]any{"name": match[1], "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
		}
		operation := map[string]any{"responses": map[string]any{"200": map[string]any{"description": "Success"}, "400": map[string]any{"description": "Invalid request"}, "401": map[string]any{"description": "Authentication required"}, "403": map[string]any{"description": "Forbidden"}, "404": map[string]any{"description": "Not found"}}, "parameters": parameters}
		if method != "GET" && method != "DELETE" {
			operation["requestBody"] = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "additionalProperties": true}}}}
		}
		item[strings.ToLower(method)] = operation
		return nil
	})
	return map[string]any{"openapi": "3.0.3", "info": map[string]any{"title": "LitLoop Go API", "version": "1.0.0", "description": "Route inventory. Request and response details are described in README.md and docs/MIGRATION.md."}, "paths": paths, "components": map[string]any{"securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}}}}
}
