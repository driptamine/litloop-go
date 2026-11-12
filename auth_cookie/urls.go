package authcookie

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/auth/login", false, s.Signin)
	s.Route("POST", "/auth/google/token", false, s.OAuth)
	s.Route("POST", "/auth/vk/token", false, s.OAuth)
	s.Route("POST", "/auth/refresh", false, s.Refresh)
	s.Route("POST", "/auth/logout", true, s.Logout)
}
