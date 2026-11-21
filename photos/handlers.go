package photos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) PhotoAlbum(q *Request) any {
	if q.R.Method == "GET" {
		var a models.PhotoAlbum
		Check(s.DB.First(&a, "friendly_token = ?", q.Param("key")).Error)
		o := Object(a)
		photos := []any{}
		for _, item := range All[models.PhotoAlbumItem](s.DB.Where("photo_album_id = ?", a.ID).Order("ordering")) {
			p := Get[models.Photo](s.DB, item.PhotoID)
			photos = append(photos, s.MediaJSON(p))
		}
		o["photos"] = photos
		o["photo_count"] = len(photos)
		album := Object(a)
		album["photo_count"] = len(photos)
		o["album"] = album
		return o
	}
	if q.Param("id") == "" {
		b := q.Body()
		a := models.PhotoAlbum{UserID: q.User.ID, Title: Truncate(Clean(Str(b["title"])), 99), Description: Clean(Str(b["description"])), FriendlyToken: Token()[:12]}
		if a.Title == "" {
			Fail(400, "title is required")
		}
		Check(s.DB.Create(&a).Error)
		q.Status = 201
		out := Object(a)
		out["photo_count"] = 0
		return out
	}
	a := Get[models.PhotoAlbum](s.DB.Where("user_id = ?", q.User.ID), q.ID("id"))
	var p models.Photo
	if strings.HasSuffix(q.R.URL.Path, "/upload") {
		file := s.MultipartUpload(q)
		p = models.Photo{UserID: Ptr(q.User.ID), Filename: file.Name, Status: "attached", FriendlyToken: Token()[:12]}
		p.R2Key, p.GCSKey, p.S3Key = file.Keys()
		Check(s.DB.Create(&p).Error)
	} else {
		p = Get[models.Photo](s.DB.Where("user_id = ?", q.User.ID), Num(q.Body()["photo_id"]))
	}
	var item models.PhotoAlbumItem
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, a.ID).Error)
		var exists int64
		Check(tx.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ? AND photo_id = ?", a.ID, p.ID).Count(&exists).Error)
		if exists > 0 {
			Fail(409, "Photo already in album")
		}
		var order int64
		Check(tx.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ?", a.ID).Select("COALESCE(MAX(ordering),0)").Scan(&order).Error)
		item = models.PhotoAlbumItem{PhotoID: p.ID, PhotoAlbumID: a.ID, Ordering: order + 1}
		return tx.Create(&item).Error
	})
	q.Status = 201
	return map[string]any{"success": true, "item_id": item.ID, "photo_id": p.ID, "album_id": a.ID, "ordering": item.Ordering, "location": s.MediaURL(p.R2Key, p.GCSKey, p.S3Key)}
}
