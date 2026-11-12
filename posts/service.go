package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func (s *Handler) LinkPost(tx *gorm.DB, p models.Post, b map[string]any, u int64) {
	for _, r := range []struct{ Key, Table, Col, Target, Owner string }{{"photo_ids", "posts_postphoto", "photo_id", "photos_photo", "user_id"}, {"video_ids", "posts_postvideo", "video_id", "videos_video", "user_id"}, {"track_ids", "posts_posttrack", "track_id", "tracks_track", "user_id"}, {"playlist_ids", "posts_postplaylist", "playlist_id", "playlists_playlist", "user_id"}} {
		v, ok := b[r.Key]
		if !ok {
			continue
		}
		list := IDs(v)
		Check(tx.Table(r.Table).Where("post_id = ?", p.ID).Delete(&map[string]any{}).Error)
		seen := map[int64]bool{}
		for i, id := range list {
			if seen[id] {
				continue
			}
			seen[id] = true
			var n int64
			Check(tx.Table(r.Target).Where("id = ? AND "+r.Owner+" = ?", id, u).Count(&n).Error)
			if n == 0 {
				Fail(400, "Media is missing or belongs to another user")
			}
			Check(tx.Table(r.Table).Create(map[string]any{"post_id": p.ID, r.Col: id, "order": i + 1}).Error)
			if r.Key != "playlist_ids" {
				Check(tx.Table(r.Target).Where("id = ?", id).Update("status", "attached").Error)
			}
		}
	}
}
