package videos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/videos/{id:[0-9]+}/encode", true, s.EncodeVideo)
	s.Route("GET", "/videos/{id:[0-9]+}/comments", false, s.Comments)
	s.Route("POST", "/videos/{id:[0-9]+}/comments/create", true, s.CommentCreate)
	s.Route("POST", "/videos/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/videos/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/videos/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/videos/r2/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/videos/r2/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/videos/r2/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/videos/gcs/create_presigned_url", true, s.UploadInitiate)
	s.Route("POST", "/videos/gcs/get_presigned_url", true, s.UploadPart)
	s.Route("POST", "/videos/gcs/complete_upload", true, s.UploadComplete)
	s.Route("POST", "/videos/gcs/native/initiate", true, s.UploadInitiate)
	s.Route("POST", "/videos/gcs/native/complete", true, s.UploadComplete)
	s.Route("POST", "/videos/create_video", true, s.UploadComplete)
	s.Route("POST", "/videos/launch-spot", true, s.LaunchSpot)
	s.Route("GET", "/videos/{id:[0-9]+}", false, func(q *Request) any { return s.MediaJSON(s.VideoAccess(q, q.ID("id"))) })
	s.Route("POST", "/videos/api/save_playback", true, s.Playback)
	s.Route("GET", "/videos/api/get_playback_time", true, s.Playback)
}
