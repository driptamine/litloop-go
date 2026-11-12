package app

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/openapi.json", false, func(q *Request) any { return s.OpenAPI() })
	s.Route("GET", "/healthz", false, func(q *Request) any { return map[string]any{"status": "ok"} })
	s.Route("GET", "/readyz", false, func(q *Request) any {
		db, e := s.DB.DB()
		Check(e)
		Check(db.PingContext(
			q.R.Context()))

		return map[string]any{"status": "ok"}
	})
}
