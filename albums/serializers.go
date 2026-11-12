package albums

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) AlbumJSON(a models.Album) any {
	o := Object(a)
	o["pk"] = a.ID
	o["id"] = a.AlbumURI
	o["images"] = All[models.Image](s.DB.Where("album_id = ?", a.ID))
	artists := []any{}
	for _, artist := range All[models.Artist](s.DB.Where("id IN (?)", s.DB.Model(&models.AlbumArtist{}).Select("artist_id").Where("album_id = ?", a.ID))) {
		artists = append(artists, s.ArtistJSON(artist))
	}
	o["artists"] = artists
	return o
}
