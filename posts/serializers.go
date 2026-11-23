package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) PostJSON(post models.Post, user *models.User) map[string]any {
	postJSON := Object(post)
	postJSON["author"] = nil
	postJSON["is_liked"] = false
	postJSON["total_likes"] = post.LikesCount
	if post.AuthorID != nil {
		var author models.User
		if handler.DB.First(&author, *post.AuthorID).Error == nil {
			postJSON["author"] = map[string]any{"id": author.ID, "username": author.Username, "avatar": author.Avatar, "is_own": user != nil && user.ID == author.ID}
		}
	}
	if user != nil {
		var count int64
		Check(handler.DB.Model(&models.PostLike{}).Where("post_id = ? AND user_id = ?", post.ID, user.ID).Count(&count).Error)
		postJSON["is_liked"] = count > 0
	}
	photos := All[models.Photo](handler.DB.Where("id IN (?)", handler.DB.Model(&models.PostPhoto{}).Select("photo_id").Where("post_id = ?", post.ID).Order("\"order\"")))
	postJSON["photos"] = handler.MediaList(photos)
	videos := All[models.Video](handler.DB.Where("id IN (?)", handler.DB.Model(&models.PostVideo{}).Select("video_id").Where("post_id = ?", post.ID).Order("\"order\"")))
	postJSON["videos"] = handler.MediaList(videos)
	tracks := All[models.Track](handler.DB.Where("id IN (?)", handler.DB.Model(&models.PostTrack{}).Select("track_id").Where("post_id = ?", post.ID)))
	out := []any{}
	for _, track := range tracks {
		out = append(out, handler.TrackJSON(track))
	}
	postJSON["tracks"] = out
	postJSON["playlists"] = All[models.Playlist](handler.DB.Where("id IN (?)", handler.DB.Model(&models.PostPlaylist{}).Select("playlist_id").Where("post_id = ?", post.ID)))
	return postJSON
}
