package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) Community(q *Request) models.Community {
	if q.Param("handle") != "" {
		var c models.Community
		Check(s.DB.First(&c, "handle = ?", q.Param("handle")).Error)
		return c
	}
	return Get[models.Community](s.DB, q.ID("id"))
}
func (s *Handler) CommunityRole(c models.Community, u *models.User, moderator bool) {
	if u == nil {
		Fail(401, "Authentication required")
	}
	if c.CreatedByID == u.ID || u.IsSuperuser {
		return
	}
	var m models.CommunityMembership
	if s.DB.First(&m, "community_id = ? AND user_id = ?", c.ID, u.ID).Error != nil {
		Fail(403, "Forbidden")
	}
	if m.Role != "admin" && !(moderator && m.Role == "moderator") {
		Fail(403, "Forbidden")
	}
}
