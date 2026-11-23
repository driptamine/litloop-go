package notifications

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/ws/notifications", false, server.Websocket)
	server.Route("GET", "/notifications", true, server.Notifications)
	server.Route("POST", "/notifications/read", true, server.Notifications)
}
