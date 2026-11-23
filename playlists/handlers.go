package playlists

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
)

func (handler *Handler) Playlist(req *Request) any {
	if req.R.Method == "POST" {
		body := req.Body()
		title := Str(body["title"])
		if title == "" {
			title = Str(body["name"])
		}
		if title == "" {
			Fail(400, "name is required")
		}
		playlist := models.Playlist{Title: Truncate(Clean(title), 99), Description: Clean(Str(body["description"])), UserID: Ptr(req.User.ID), FriendlyToken: Token()[:12]}
		Check(handler.DB.Create(&playlist).Error)
		req.Status = 201
		return handler.PlaylistJSON(playlist)
	}
	if req.Param("key") != "" {
		var playlist models.Playlist
		key := req.Param("key")
		if parsed, err := strconv.ParseInt(key, 10, 64); err == nil {
			Check(handler.DB.First(&playlist, parsed).Error)
		} else {
			Check(handler.DB.First(&playlist, "friendly_token = ?", key).Error)
		}
		return handler.PlaylistJSON(playlist)
	}
	db := handler.DB.Model(&models.Playlist{})
	if author := req.R.URL.Query().Get("author"); author != "" {
		db = db.Where("user_id IN (?)", handler.DB.Model(&models.User{}).Select("id").Where("username = ?", author))
	}
	result := req.Page(db.Order("id DESC"), &[]models.Playlist{}, func(value any) any {
		out := []any{}
		for _, playlist := range *value.(*[]models.Playlist) {
			out = append(out, handler.PlaylistJSON(playlist))
		}
		return out
	}).(map[string]any)
	result["results"] = result["items"]
	result["count"] = result["total"]
	return result
}
