package artists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) ArtistJSON(a models.Artist) any {
	o := Object(a)
	o["pk"] = a.ID
	o["id"] = a.ArtistURI
	o["images"] = All[models.Image](s.DB.Where("artist_id = ?", a.ID))
	return o
}
