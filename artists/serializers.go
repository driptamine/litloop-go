package artists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) ArtistJSON(artist models.Artist) any {
	out := Object(artist)
	out["pk"] = artist.ID
	out["id"] = artist.ArtistURI
	out["images"] = All[models.Image](handler.DB.Where("artist_id = ?", artist.ID))
	return out
}
