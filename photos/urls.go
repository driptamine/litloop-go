package photos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/photos/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/photos/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/photos/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/photos/r2/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/photos/r2/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/photos/r2/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/photos/gcs/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/photos/gcs/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/photos/gcs/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/photos/album/create", true, s.PhotoAlbum)
	s.Route("GET", "/photos/album/{key}", false, s.PhotoAlbum)
	s.Route("POST", "/photos/album/{id:[0-9]+}/add", true, s.PhotoAlbum)
	s.Route("POST", "/photos/album/{id:[0-9]+}/upload", true, s.PhotoAlbum)
}
