package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) CommunityJSON(c models.Community, u *models.User) map[string]any {
	o := Object(c)
	var count int64
	Check(s.DB.Model(&models.CommunityMembership{}).Where("community_id = ?", c.ID).Count(&count).Error)
	o["members_count"] = count
	o["member_count"] = count
	o["created_by"] = c.CreatedByID
	creator := Get[models.User](s.DB, c.CreatedByID)
	o["created_by_username"] = creator.Username
	o["is_member"] = false
	o["role"] = nil
	if u != nil {
		var m models.CommunityMembership
		if s.DB.First(&m, "community_id = ? AND user_id = ?", c.ID, u.ID).Error == nil {
			o["is_member"] = true
			o["role"] = m.Role
		}
	}
	o["user_is_member"] = o["is_member"]
	o["user_role"] = o["role"]
	return o
}
func (s *Handler) CommunityPostJSON(cp models.CommunityPost, u *models.User) map[string]any {
	o := s.PostJSON(Get[models.Post](s.DB, cp.PostID), u)
	c := Get[models.Community](s.DB, cp.CommunityID)
	o["community_id"] = c.ID
	o["community_name"] = c.Name
	o["status"] = cp.Status
	o["requested_by"] = PublicUser(Get[models.User](s.DB, cp.RequestedByID))
	o["reviewed_by"] = nil
	o["reviewed_at"] = cp.ReviewedAt
	if cp.ReviewedByID != nil {
		o["reviewed_by"] = PublicUser(Get[models.User](s.DB, *cp.ReviewedByID))
	}
	o["community_post_id"] = cp.ID
	return o
}
