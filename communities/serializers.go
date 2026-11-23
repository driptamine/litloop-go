package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) CommunityJSON(community models.Community, user *models.User) map[string]any {
	out := Object(community)
	var count int64
	Check(handler.DB.Model(&models.CommunityMembership{}).Where("community_id = ?", community.ID).Count(&count).Error)
	out["members_count"] = count
	out["member_count"] = count
	out["created_by"] = community.CreatedByID
	creator := Get[models.User](handler.DB, community.CreatedByID)
	out["created_by_username"] = creator.Username
	out["is_member"] = false
	out["role"] = nil
	if user != nil {
		var membership models.CommunityMembership
		if handler.DB.First(&membership, "community_id = ? AND user_id = ?", community.ID, user.ID).Error == nil {
			out["is_member"] = true
			out["role"] = membership.Role
		}
	}
	out["user_is_member"] = out["is_member"]
	out["user_role"] = out["role"]
	return out
}
func (handler *Handler) CommunityPostJSON(cp models.CommunityPost, user *models.User) map[string]any {
	out := handler.PostJSON(Get[models.Post](handler.DB, cp.PostID), user)
	community := Get[models.Community](handler.DB, cp.CommunityID)
	out["community_id"] = community.ID
	out["community_name"] = community.Name
	out["status"] = cp.Status
	out["requested_by"] = PublicUser(Get[models.User](handler.DB, cp.RequestedByID))
	out["reviewed_by"] = nil
	out["reviewed_at"] = cp.ReviewedAt
	if cp.ReviewedByID != nil {
		out["reviewed_by"] = PublicUser(Get[models.User](handler.DB, *cp.ReviewedByID))
	}
	out["community_post_id"] = cp.ID
	return out
}
