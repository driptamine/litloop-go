package playlists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) PlaylistJSON(playlist models.Playlist) map[string]any {
	out := Object(playlist)
	out["name"] = playlist.Title
	out["user"] = nil
	if playlist.UserID != nil {
		out["user"] = Get[models.User](handler.DB, *playlist.UserID).Username
	}
	return out
}
