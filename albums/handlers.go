package albums

import (
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) AlbumLike(req *Request) any {
	var album models.Album
	Check(handler.DB.First(&album, "album_uri = ?", req.Param("key")).Error)
	var liked bool
	var count int64
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&album, album.ID).Error)
		var like models.AlbumLike
		err := tx.First(&like, "album_id = ? AND liked_by_id = ?", album.ID, req.User.ID).Error
		if err == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.AlbumLike{AlbumID: album.ID, LikedByID: req.User.ID}).Error)
			liked = true
		} else {
			Check(err)
			Check(tx.Delete(&like).Error)
		}
		Check(tx.Model(&models.AlbumLike{}).Where("album_id = ?", album.ID).Count(&count).Error)
		return engagement.EnqueueReaction(tx, handler.Config, "album", album.ID, req.User.ID, liked)
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
