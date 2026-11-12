package albums

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) AlbumLike(q *Request) any {
	var a models.Album
	Check(s.DB.First(&a, "album_uri = ?", q.Param("key")).Error)
	var liked bool
	var count int64
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, a.ID).Error)
		var l models.AlbumLike
		e := tx.First(&l, "album_id = ? AND liked_by_id = ?", a.ID, q.User.ID).Error
		if e == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.AlbumLike{AlbumID: a.ID, LikedByID: q.User.ID}).Error)
			liked = true
		} else {
			Check(e)
			Check(tx.Delete(&l).Error)
		}
		return tx.Model(&models.AlbumLike{}).Where("album_id = ?", a.ID).Count(&count).Error
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
