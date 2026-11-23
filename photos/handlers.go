package photos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) PhotoAlbum(req *Request) any {
	if req.R.Method == "GET" {
		var albumData models.PhotoAlbum
		Check(handler.DB.First(&albumData, "friendly_token = ?", req.Param("key")).Error)
		out := Object(albumData)
		photos := []any{}
		for _, item := range All[models.PhotoAlbumItem](handler.DB.Where("photo_album_id = ?", albumData.ID).Order("ordering")) {
			photo := Get[models.Photo](handler.DB, item.PhotoID)
			photos = append(photos, handler.MediaJSON(photo))
		}
		out["photos"] = photos
		out["photo_count"] = len(photos)
		album := Object(albumData)
		album["photo_count"] = len(photos)
		out["album"] = album
		return out
	}
	if req.Param("id") == "" {
		body := req.Body()
		album := models.PhotoAlbum{UserID: req.User.ID, Title: Truncate(Clean(Str(body["title"])), 99), Description: Clean(Str(body["description"])), FriendlyToken: Token()[:12]}
		if album.Title == "" {
			Fail(400, "title is required")
		}
		Check(handler.DB.Create(&album).Error)
		req.Status = 201
		out := Object(album)
		out["photo_count"] = 0
		return out
	}
	album := Get[models.PhotoAlbum](handler.DB.Where("user_id = ?", req.User.ID), req.ID("id"))
	var photo models.Photo
	if strings.HasSuffix(req.R.URL.Path, "/upload") {
		file := handler.MultipartUpload(req)
		photo = models.Photo{UserID: Ptr(req.User.ID), Filename: file.Name, Status: "attached", FriendlyToken: Token()[:12]}
		photo.R2Key, photo.GCSKey, photo.S3Key = file.Keys()
		Check(handler.DB.Create(&photo).Error)
	} else {
		photo = Get[models.Photo](handler.DB.Where("user_id = ?", req.User.ID), Num(req.Body()["photo_id"]))
	}
	var item models.PhotoAlbumItem
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&album, album.ID).Error)
		var exists int64
		Check(tx.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ? AND photo_id = ?", album.ID, photo.ID).Count(&exists).Error)
		if exists > 0 {
			Fail(409, "Photo already in album")
		}
		var order int64
		Check(tx.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ?", album.ID).Select("COALESCE(MAX(ordering),0)").Scan(&order).Error)
		item = models.PhotoAlbumItem{PhotoID: photo.ID, PhotoAlbumID: album.ID, Ordering: order + 1}
		return tx.Create(&item).Error
	})
	req.Status = 201
	return map[string]any{"success": true, "item_id": item.ID, "photo_id": photo.ID, "album_id": album.ID, "ordering": item.Ordering, "location": handler.MediaURL(photo.R2Key, photo.GCSKey, photo.S3Key)}
}
