package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) PostJSON(p models.Post, u *models.User) map[string]any {
	o := Object(p)
	o["author"] = nil
	o["is_liked"] = false
	o["total_likes"] = p.LikesCount
	if p.AuthorID != nil {
		var author models.User
		if s.DB.First(&author, *p.AuthorID).Error == nil {
			o["author"] = map[string]any{"id": author.ID, "username": author.Username, "avatar": author.Avatar, "is_own": u != nil && u.ID == author.ID}
		}
	}
	if u != nil {
		var n int64
		Check(s.DB.Model(&models.PostLike{}).Where("post_id = ? AND user_id = ?", p.ID, u.ID).Count(&n).Error)
		o["is_liked"] = n > 0
	}
	photos := All[models.Photo](s.DB.Where("id IN (?)", s.DB.Model(&models.PostPhoto{}).Select("photo_id").Where("post_id = ?", p.ID).Order("\"order\"")))
	o["photos"] = s.MediaList(photos)
	videos := All[models.Video](s.DB.Where("id IN (?)", s.DB.Model(&models.PostVideo{}).Select("video_id").Where("post_id = ?", p.ID).Order("\"order\"")))
	o["videos"] = s.MediaList(videos)
	tracks := All[models.Track](s.DB.Where("id IN (?)", s.DB.Model(&models.PostTrack{}).Select("track_id").Where("post_id = ?", p.ID)))
	out := []any{}
	for _, t := range tracks {
		out = append(out, s.TrackJSON(t))
	}
	o["tracks"] = out
	o["playlists"] = All[models.Playlist](s.DB.Where("id IN (?)", s.DB.Model(&models.PostPlaylist{}).Select("playlist_id").Where("post_id = ?", p.ID)))
	return o
}
