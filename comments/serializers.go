package comments

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) CommentJSON(c models.Comment) map[string]any {
	o := Object(c)
	u := Get[models.User](s.DB, c.UserID)
	o["username"] = u.Username
	o["avatar"] = u.Avatar
	o["created_at"] = c.AddDate
	o["author_name"] = u.Username
	o["author_thumbnail_url"] = u.Avatar
	o["author_profile"] = "/users/" + u.Username + "/"
	o["parent"] = nil
	if c.ParentID != nil {
		var parent models.Comment
		if s.DB.First(&parent, *c.ParentID).Error == nil {
			o["parent"] = parent.UID
		}
	}
	return o
}
