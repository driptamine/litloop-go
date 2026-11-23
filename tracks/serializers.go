package tracks

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) TrackJSON(track models.Track) any {
	out := handler.MediaJSON(track)
	out["pk"] = track.ID
	out["id"] = track.TrackURI
	out["user"] = nil
	if track.UserID != nil {
		var user models.User
		if handler.DB.First(&user, *track.UserID).Error == nil {
			out["user"] = PublicUser(user)
		}
	}
	artists := []any{}
	for _, artist := range All[models.Artist](handler.DB.Where("id IN (?)", handler.DB.Model(&models.TrackArtist{}).Select("artist_id").Where("track_id = ?", track.ID))) {
		artists = append(artists, handler.ArtistJSON(artist))
	}
	out["artists"] = artists
	return out
}
