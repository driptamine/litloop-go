package comments

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) CommentJSON(comment models.Comment) map[string]any {
	out := Object(comment)
	user := Get[models.User](handler.DB, comment.UserID)
	out["username"] = user.Username
	out["avatar"] = user.Avatar
	out["created_at"] = comment.AddDate
	out["author_name"] = user.Username
	out["author_thumbnail_url"] = user.Avatar
	out["author_profile"] = "/users/" + user.Username + "/"
	out["parent"] = nil
	if comment.ParentID != nil {
		var parent models.Comment
		if handler.DB.First(&parent, *comment.ParentID).Error == nil {
			out["parent"] = parent.UID
		}
	}
	return out
}
