package tracks

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) TrackJSON(t models.Track) any {
	o := s.MediaJSON(t)
	o["pk"] = t.ID
	o["id"] = t.TrackURI
	o["user"] = nil
	if t.UserID != nil {
		var u models.User
		if s.DB.First(&u, *t.UserID).Error == nil {
			o["user"] = PublicUser(u)
		}
	}
	artists := []any{}
	for _, a := range All[models.Artist](s.DB.Where("id IN (?)", s.DB.Model(&models.TrackArtist{}).Select("artist_id").Where("track_id = ?", t.ID))) {
		artists = append(artists, s.ArtistJSON(a))
	}
	o["artists"] = artists
	return o
}
