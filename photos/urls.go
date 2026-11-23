package photos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/photos/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/photos/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/photos/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/photos/r2/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/photos/r2/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/photos/r2/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/photos/gcs/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/photos/gcs/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/photos/gcs/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/photos/album/create", true, server.PhotoAlbum)
	server.Route("GET", "/photos/album/{key}", false, server.PhotoAlbum)
	server.Route("POST", "/photos/album/{id:[0-9]+}/add", true, server.PhotoAlbum)
	server.Route("POST", "/photos/album/{id:[0-9]+}/upload", true, server.PhotoAlbum)
}
