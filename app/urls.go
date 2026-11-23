package app

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/openapi.json", false, func(req *Request) any { return server.OpenAPI() })
	server.Route("GET", "/healthz", false, func(req *Request) any { return map[string]any{"status": "ok"} })
	server.Route("GET", "/readyz", false, func(req *Request) any {
		db, err := server.DB.DB()
		Check(err)
		Check(db.PingContext(
			req.R.Context()))

		return map[string]any{"status": "ok"}
	})
}
