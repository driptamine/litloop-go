package albums

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/album", false, s.Catalog)
	s.Route("GET", "/album/all", false, s.Catalog)
	s.Route("GET", "/album/search", false, s.Catalog)
	s.Route("GET", "/album/feed", false, s.Catalog)
	s.Route("GET", "/album/{key}/upd", false, s.Catalog)
	s.Route("GET", "/album/{key}", false, s.Catalog)
	s.Route("POST,PUT,DELETE", "/album/{key}/fike", true, s.AlbumLike)
	s.Route("POST", "/album/{id:[0-9]+}/view", true, s.View)
}
