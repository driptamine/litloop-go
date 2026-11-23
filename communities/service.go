package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) Community(req *Request) models.Community {
	if req.Param("handle") != "" {
		var community models.Community
		Check(handler.DB.First(&community, "handle = ?", req.Param("handle")).Error)
		return community
	}
	return Get[models.Community](handler.DB, req.ID("id"))
}
func (handler *Handler) CommunityRole(community models.Community, user *models.User, moderator bool) {
	if user == nil {
		Fail(401, "Authentication required")
	}
	if community.CreatedByID == user.ID || user.IsSuperuser {
		return
	}
	var membership models.CommunityMembership
	if handler.DB.First(&membership, "community_id = ? AND user_id = ?", community.ID, user.ID).Error != nil {
		Fail(403, "Forbidden")
	}
	if membership.Role != "admin" && !(moderator && membership.Role == "moderator") {
		Fail(403, "Forbidden")
	}
}
