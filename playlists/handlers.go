package playlists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
)

func (s *Handler) Playlist(q *Request) any {
	if q.R.Method == "POST" {
		b := q.Body()
		title := Str(b["title"])
		if title == "" {
			title = Str(b["name"])
		}
		if title == "" {
			Fail(400, "name is required")
		}
		p := models.Playlist{Title: Truncate(Clean(title), 99), Description: Clean(Str(b["description"])), UserID: Ptr(q.User.ID), FriendlyToken: Token()[:12]}
		Check(s.DB.Create(&p).Error)
		q.Status = 201
		return s.PlaylistJSON(p)
	}
	if q.Param("key") != "" {
		var p models.Playlist
		key := q.Param("key")
		if n, e := strconv.ParseInt(key, 10, 64); e == nil {
			Check(s.DB.First(&p, n).Error)
		} else {
			Check(s.DB.First(&p, "friendly_token = ?", key).Error)
		}
		return s.PlaylistJSON(p)
	}
	db := s.DB.Model(&models.Playlist{})
	if author := q.R.URL.Query().Get("author"); author != "" {
		db = db.Where("user_id IN (?)", s.DB.Model(&models.User{}).Select("id").Where("username = ?", author))
	}
	result := q.Page(db.Order("id DESC"), &[]models.Playlist{}, func(v any) any {
		out := []any{}
		for _, p := range *v.(*[]models.Playlist) {
			out = append(out, s.PlaylistJSON(p))
		}
		return out
	}).(map[string]any)
	result["results"] = result["items"]
	result["count"] = result["total"]
	return result
}
