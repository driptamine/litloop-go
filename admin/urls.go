package admin

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/admin", true, s.Admin)
	s.Route("GET,POST", "/admin/{table}", true, s.Admin)
	s.Route("GET,PUT,PATCH,DELETE", "/admin/{table}/{id}", true, s.Admin)
	s.Route("POST", "/maintenance/delete-tables", true, s.Maintenance)
	s.Route("POST", "/maintenance/regenerate", true, s.Maintenance)
	s.Route("POST", "/maintenance/delete-user-tables", true, s.Maintenance)
	s.Route("POST", "/maintenance/regenerate-user-tables", true, s.Maintenance)
	s.Route("POST", "/maintenance/delete-chat-tables", true, s.Maintenance)
	s.Route("POST", "/maintenance/regenerate-chat-tables", true, s.Maintenance)
}
