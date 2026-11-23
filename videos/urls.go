package videos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/videos/{id:[0-9]+}/encode", true, server.EncodeVideo)
	server.Route("GET", "/videos/{id:[0-9]+}/comments", false, server.Comments)
	server.Route("POST", "/videos/{id:[0-9]+}/comments/create", true, server.CommentCreate)
	server.Route("POST", "/videos/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/videos/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/videos/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/videos/r2/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/videos/r2/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/videos/r2/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/videos/gcs/create_presigned_url", true, server.UploadInitiate)
	server.Route("POST", "/videos/gcs/get_presigned_url", true, server.UploadPart)
	server.Route("POST", "/videos/gcs/complete_upload", true, server.UploadComplete)
	server.Route("POST", "/videos/gcs/native/initiate", true, server.UploadInitiate)
	server.Route("POST", "/videos/gcs/native/complete", true, server.UploadComplete)
	server.Route("POST", "/videos/create_video", true, server.UploadComplete)
	server.Route("POST", "/videos/launch-spot", true, server.LaunchSpot)
	server.Route("GET", "/videos/{id:[0-9]+}", false, func(req *Request) any { return server.MediaJSON(server.VideoAccess(req, req.ID("id"))) })
	server.Route("POST", "/videos/api/save_playback", true, server.Playback)
	server.Route("GET", "/videos/api/get_playback_time", true, server.Playback)
}
