package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func (handler *Handler) LinkPost(tx *gorm.DB, post models.Post, body map[string]any, userID int64) {
	for _, field := range []struct{ Key, Table, Col, Target, Owner string }{{"photo_ids", "posts_postphoto", "photo_id", "photos_photo", "user_id"}, {"video_ids", "posts_postvideo", "video_id", "videos_video", "user_id"}, {"track_ids", "posts_posttrack", "track_id", "tracks_track", "user_id"}, {"playlist_ids", "posts_postplaylist", "playlist_id", "playlists_playlist", "user_id"}} {
		value, ok := body[field.Key]
		if !ok {
			continue
		}
		list := IDs(value)
		Check(tx.Table(field.Table).Where("post_id = ?", post.ID).Delete(&map[string]any{}).Error)
		seen := map[int64]bool{}
		for index, id := range list {
			if seen[id] {
				continue
			}
			seen[id] = true
			var count int64
			Check(tx.Table(field.Target).Where("id = ? AND "+field.Owner+" = ?", id, userID).Count(&count).Error)
			if count == 0 {
				Fail(400, "Media is missing or belongs to another user")
			}
			Check(tx.Table(field.Table).Create(map[string]any{"post_id": post.ID, field.Col: id, "order": index + 1}).Error)
			if field.Key != "playlist_ids" {
				Check(tx.Table(field.Target).Where("id = ?", id).Update("status", "attached").Error)
			}
		}
	}
}
