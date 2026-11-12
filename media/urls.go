package media

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/assets/{asset}", false, s.Asset)
	s.Route("GET", "/gcs-media/*", false, s.NativeMedia)
	s.Route("PUT", "/uploads/{upload}", true, s.UploadPut)
	s.Route("GET", "/media/*", false, s.MediaServe)
}
