package tracks

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) FindTrack(key string) models.Track {
	var t models.Track
	if n, e := strconv.ParseInt(key, 10, 64); e == nil {
		Check(s.DB.First(&t, n).Error)
	} else {
		Check(s.DB.First(&t, "track_uri = ?", key).Error)
	}
	return t
}
func (s *Handler) TrackWrite(q *Request) any {
	t := s.FindTrack(q.Param("key"))
	if !q.User.IsStaff {
		Owned(Val(t.UserID), q.User.ID)
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "tracks_track", t.ID) })
		q.Status = 204
		return nil
	}
	b := q.Body()
	for _, p := range []struct {
		Key   string
		Value *string
	}{{"name", &t.Name}, {"track_number", &t.TrackNumber}, {"status", &t.Status}} {
		if v, ok := b[p.Key]; ok {
			*p.Value = Str(v)
		}
	}
	if t.Status != "draft" && t.Status != "attached" {
		Fail(400, "Invalid status")
	}
	if v, ok := b["release_date"]; ok {
		t.ReleaseDate = nil
		if v != nil && Str(v) != "" {
			t.ReleaseDate = ParseDate(Str(v))
			if t.ReleaseDate == nil {
				Fail(400, "Invalid release_date")
			}
		}
	}
	q.Tx(func(tx *gorm.DB) error {
		if uri, ok := b["album_uri"]; ok {
			if uri == nil || Str(uri) == "" {
				t.AlbumID = nil
			} else {
				var a models.Album
				Check(tx.First(&a, "album_uri = ?", Str(uri)).Error)
				t.AlbumID = Ptr(a.ID)
			}
		}
		names, hasNames := b["artist_names"]
		if !hasNames {
			names, hasNames = b["artists"]
		}
		_, hasURIs := b["artist_uris"]
		if hasNames || hasURIs {
			Check(tx.Where("track_id = ?", t.ID).Delete(&models.TrackArtist{}).Error)
			artists := []models.Artist{}
			if hasURIs {
				uris, ok := b["artist_uris"].([]any)
				if !ok {
					Fail(400, "artist_uris must be an array")
				}
				list := []string{}
				for _, uri := range uris {
					list = append(list, Str(uri))
				}
				artists = All[models.Artist](tx.Where("artist_uri IN ?", list))
			}
			if hasNames {
				list := []string{}
				switch value := names.(type) {
				case string:
					separator := ","
					if strings.Contains(value, ";") {
						separator = ";"
					}
					list = strings.Split(value, separator)
				case []any:
					for _, name := range value {
						list = append(list, Str(name))
					}
				default:
					Fail(400, "artists must be a string or array")
				}
				artists = []models.Artist{}
				for _, name := range list {
					name = strings.TrimSpace(name)
					if name == "" {
						continue
					}
					a := models.Artist{Name: name, ArtistURI: "local-" + strings.Trim(SlugRE.ReplaceAllString(strings.ToLower(name), "-"), "-")}
					Check(tx.Where("name = ?", name).FirstOrCreate(&a).Error)
					artists = append(artists, a)
				}
			}
			for _, artist := range artists {
				Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.TrackArtist{TrackID: t.ID, ArtistID: artist.ID}).Error)
			}
		}
		return tx.Save(&t).Error
	})
	return s.TrackJSON(t)
}
