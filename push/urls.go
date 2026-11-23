package push

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/push/vapid-public-key", false, server.PushVAPIDKey)
	server.Route("POST", "/push/subscribe", true, server.PushSubscribe)
	server.Route("POST", "/push/unsubscribe", true, server.PushUnsubscribe)
}
