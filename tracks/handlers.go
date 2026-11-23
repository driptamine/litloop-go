package tracks

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) FindTrack(key string) models.Track {
	var track models.Track
	if parsed, err := strconv.ParseInt(key, 10, 64); err == nil {
		Check(handler.DB.First(&track, parsed).Error)
	} else {
		Check(handler.DB.First(&track, "track_uri = ?", key).Error)
	}
	return track
}
func (handler *Handler) TrackWrite(req *Request) any {
	track := handler.FindTrack(req.Param("key"))
	if !req.User.IsStaff {
		Owned(Val(track.UserID), req.User.ID)
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "tracks_track", track.ID) })
		req.Status = 204
		return nil
	}
	body := req.Body()
	for _, field := range []struct {
		Key	string
		Value	*string
	}{{"name", &track.Name}, {"track_number", &track.TrackNumber}, {"status", &track.Status}} {
		if value, ok := body[field.Key]; ok {
			*field.Value = Str(value)
		}
	}
	if track.Status != "draft" && track.Status != "attached" {
		Fail(400, "Invalid status")
	}
	if value, ok := body["release_date"]; ok {
		track.ReleaseDate = nil
		if value != nil && Str(value) != "" {
			track.ReleaseDate = ParseDate(Str(value))
			if track.ReleaseDate == nil {
				Fail(400, "Invalid release_date")
			}
		}
	}
	req.Tx(func(tx *gorm.DB) error {
		if uri, ok := body["album_uri"]; ok {
			if uri == nil || Str(uri) == "" {
				track.AlbumID = nil
			} else {
				var album models.Album
				Check(tx.First(&album, "album_uri = ?", Str(uri)).Error)
				track.AlbumID = Ptr(album.ID)
			}
		}
		names, hasNames := body["artist_names"]
		if !hasNames {
			names, hasNames = body["artists"]
		}
		_, hasURIs := body["artist_uris"]
		if hasNames || hasURIs {
			Check(tx.Where("track_id = ?", track.ID).Delete(&models.TrackArtist{}).Error)
			artists := []models.Artist{}
			if hasURIs {
				uris, ok := body["artist_uris"].([]any)
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
					artist := models.Artist{Name: name, ArtistURI: "local-" + strings.Trim(SlugRE.ReplaceAllString(strings.ToLower(name), "-"), "-")}
					Check(tx.Where("name = ?", name).FirstOrCreate(&artist).Error)
					artists = append(artists, artist)
				}
			}
			for _, artist := range artists {
				Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.TrackArtist{TrackID: track.ID, ArtistID: artist.ID}).Error)
			}
		}
		return tx.Save(&track).Error
	})
	return handler.TrackJSON(track)
}
