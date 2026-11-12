package catalog

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Catalog(q *Request) any {
	kind := strings.Split(strings.TrimPrefix(q.R.URL.Path, "/"), "/")[0]
	key := q.Param("key")
	term := q.R.URL.Query().Get("q")
	suffix := q.R.URL.Path
	var db *gorm.DB
	var dest any
	var encode func(any) any
	switch kind {
	case "artist":
		db = s.DB.Model(&models.Artist{})
		dest = &[]models.Artist{}
		encode = func(v any) any {
			out := []any{}
			for _, a := range *v.(*[]models.Artist) {
				out = append(out, s.ArtistJSON(a))
			}
			return out
		}
		if key != "" {
			var a models.Artist
			Check(s.DB.First(&a, "artist_uri = ?", key).Error)
			if strings.Contains(suffix, "/albums") {
				out := []any{}
				for _, album := range All[models.Album](s.DB.Where("id IN (?)", s.DB.Model(&models.AlbumArtist{}).Select("album_id").Where("artist_id = ?", a.ID))) {
					out = append(out, s.AlbumJSON(album))
				}
				return map[string]any{"albums": out}
			}
			return s.ArtistJSON(a)
		}
	case "album":
		db = s.DB.Model(&models.Album{})
		dest = &[]models.Album{}
		encode = func(v any) any {
			out := []any{}
			for _, a := range *v.(*[]models.Album) {
				out = append(out, s.AlbumJSON(a))
			}
			return out
		}
		if key != "" {
			var a models.Album
			Check(s.DB.First(&a, "album_uri = ?", key).Error)
			return s.AlbumJSON(a)
		}
	case "tracks":
		db = s.DB.Model(&models.Track{})
		dest = &[]models.Track{}
		encode = func(v any) any {
			out := []any{}
			for _, t := range *v.(*[]models.Track) {
				out = append(out, s.TrackJSON(t))
			}
			return out
		}
		if key != "" {
			return s.TrackJSON(s.FindTrack(key))
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
	page := q.Page(db, dest, encode).(map[string]any)
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
