package admin

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/admin", true, server.Admin)
	server.Route("GET,POST", "/admin/{table}", true, server.Admin)
	server.Route("GET,PUT,PATCH,DELETE", "/admin/{table}/{id}", true, server.Admin)
	server.Route("POST", "/maintenance/delete-tables", true, server.Maintenance)
	server.Route("POST", "/maintenance/regenerate", true, server.Maintenance)
	server.Route("POST", "/maintenance/delete-user-tables", true, server.Maintenance)
	server.Route("POST", "/maintenance/regenerate-user-tables", true, server.Maintenance)
	server.Route("POST", "/maintenance/delete-chat-tables", true, server.Maintenance)
	server.Route("POST", "/maintenance/regenerate-chat-tables", true, server.Maintenance)
}
