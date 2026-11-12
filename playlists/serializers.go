package playlists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) PlaylistJSON(p models.Playlist) map[string]any {
	o := Object(p)
	o["name"] = p.Title
	o["user"] = nil
	if p.UserID != nil {
		o["user"] = Get[models.User](s.DB, *p.UserID).Username
	}
	return o
}
