package catalog

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Catalog(req *Request) any {
	kind := strings.Split(strings.TrimPrefix(req.R.URL.Path, "/"), "/")[0]
	key := req.Param("key")
	term := req.R.URL.Query().Get("q")
	suffix := req.R.URL.Path
	var db *gorm.DB
	var dest any
	var encode func(any) any
	switch kind {
	case "artist":
		db = handler.DB.Model(&models.Artist{})
		dest = &[]models.Artist{}
		encode = func(value any) any {
			out := []any{}
			for _, artist := range *value.(*[]models.Artist) {
				out = append(out, handler.ArtistJSON(artist))
			}
			return out
		}
		if key != "" {
			var album models.Artist
			Check(handler.DB.First(&album, "artist_uri = ?", key).Error)
			if strings.Contains(suffix, "/albums") {
				out := []any{}
				for _, album := range All[models.Album](handler.DB.Where("id IN (?)", handler.DB.Model(&models.AlbumArtist{}).Select("album_id").Where("artist_id = ?", album.ID))) {
					out = append(out, handler.AlbumJSON(album))
				}
				return map[string]any{"albums": out}
			}
			return handler.ArtistJSON(album)
		}
	case "album":
		db = handler.DB.Model(&models.Album{})
		dest = &[]models.Album{}
		encode = func(value any) any {
			out := []any{}
			for _, album := range *value.(*[]models.Album) {
				out = append(out, handler.AlbumJSON(album))
			}
			return out
		}
		if key != "" {
			var album models.Album
			Check(handler.DB.First(&album, "album_uri = ?", key).Error)
			return handler.AlbumJSON(album)
		}
	case "tracks":
		db = handler.DB.Model(&models.Track{})
		dest = &[]models.Track{}
		encode = func(value any) any {
			out := []any{}
			for _, track := range *value.(*[]models.Track) {
				out = append(out, handler.TrackJSON(track))
			}
			return out
		}
		if key != "" {
			return handler.TrackJSON(handler.FindTrack(key))
		}
	default:
		Fail(404, "Not found")
	}
	if term != "" {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(term)+"%")
	}
	db = db.Order("id")
	if strings.HasSuffix(suffix, "/all") || strings.HasSuffix(suffix, "/search") {
		Check(db.Limit(1000).Find(dest).Error)
		if strings.HasSuffix(suffix, "/search") {
			return map[string]any{"items": encode(dest)}
		}
		name := kind + "s"
		if kind == "tracks" {
			name = "tracks"
		}
		return map[string]any{"return_code": "0000", "return_message": "Success", name: encode(dest)}
	}
	page := req.Page(db, dest, encode).(map[string]any)
	if strings.HasSuffix(suffix, "/feed") {
		return page
	}
	name := kind + "s"
	if kind == "tracks" {
		name = "tracks"
	}
	page[name] = page["items"]
	delete(page, "items")
	page["return_code"] = "0000"
	page["return_message"] = "Success"
	return page
}
