package albums

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) AlbumJSON(album models.Album) any {
	out := Object(album)
	out["pk"] = album.ID
	out["id"] = album.AlbumURI
	out["images"] = All[models.Image](handler.DB.Where("album_id = ?", album.ID))
	artists := []any{}
	for _, artist := range All[models.Artist](handler.DB.Where("id IN (?)", handler.DB.Model(&models.AlbumArtist{}).Select("artist_id").Where("album_id = ?", album.ID))) {
		artists = append(artists, handler.ArtistJSON(artist))
	}
	out["artists"] = artists
	return out
}
