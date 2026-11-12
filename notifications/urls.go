package notifications

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/ws/notifications", false, s.Websocket)
	s.Route("GET", "/notifications", true, s.Notifications)
	s.Route("POST", "/notifications/read", true, s.Notifications)
}
