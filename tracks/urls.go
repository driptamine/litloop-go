package tracks

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/tracks/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/tracks/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/tracks/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/tracks/r2/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/tracks/r2/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/tracks/r2/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/tracks/gcs/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/tracks/gcs/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/tracks/gcs/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/tracks/gcs/native/initiate", true, s.UploadInitiate)
	s.Route("POST", "/tracks/gcs/native/complete", true, s.UploadComplete)
	s.Route("GET", "/tracks", false, s.Catalog)
	s.Route("GET", "/tracks/all", false, s.Catalog)
	s.Route("GET", "/tracks/search", false, s.Catalog)
	s.Route("GET", "/tracks/{key}", false, s.Catalog)
	s.Route("GET,PUT,PATCH", "/tracks/{key}/upd", true, s.TrackWrite)
	s.Route("DELETE", "/tracks/{key}/del", true, s.TrackWrite)
}
