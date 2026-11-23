package media

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/assets/{asset}", false, server.Asset)
	server.Route("GET", "/gcs-media/*", false, server.NativeMedia)
	server.Route("PUT", "/uploads/{upload}", true, server.UploadPut)
	server.Route("GET", "/media/*", false, server.MediaServe)
}
