package authcookie

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/auth/login", false, server.Signin)
	server.Route("POST", "/auth/google/token", false, server.OAuth)
	server.Route("POST", "/auth/vk/token", false, server.OAuth)
	server.Route("POST", "/auth/refresh", false, server.Refresh)
	server.Route("POST", "/auth/logout", true, server.Logout)
}
