package tracks

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/tracks/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/tracks/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/tracks/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/tracks/r2/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/tracks/r2/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/tracks/r2/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/tracks/gcs/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/tracks/gcs/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/tracks/gcs/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/tracks/gcs/native/initiate", true, server.UploadInitiate)
	server.Route("POST", "/tracks/gcs/native/complete", true, server.UploadComplete)
	server.Route("GET", "/tracks", false, server.Catalog)
	server.Route("GET", "/tracks/all", false, server.Catalog)
	server.Route("GET", "/tracks/search", false, server.Catalog)
	server.Route("GET", "/tracks/{key}", false, server.Catalog)
	server.Route("GET,PUT,PATCH", "/tracks/{key}/upd", true, server.TrackWrite)
	server.Route("DELETE", "/tracks/{key}/del", true, server.TrackWrite)
}
