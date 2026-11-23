package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) Communities(req *Request) any {
	out := []any{}
	db := handler.DB.Order("id DESC")
	if Search := req.R.URL.Query().Get("q"); Search != "" {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(Search)+"%")
	}
	for _, community := range All[models.Community](db) {
		out = append(out, handler.CommunityJSON(community, req.User))
	}
	return map[string]any{"communities": out}
}
func (handler *Handler) CommunityWrite(req *Request) any {
	var community models.Community
	creating := req.Param("id") == ""
	if !creating {
		community = handler.Community(req)
		handler.CommunityRole(community, req.User, false)
	} else {
		community.CreatedByID = req.User.ID
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "communities_community", community.ID) })
		return map[string]any{"deleted": true, "community_id": community.ID}
	}
	body := req.Body()
	for _, field := range []struct {
		Key	string
		Value	*string
	}{{"name", &community.Name}, {"description", &community.Description}, {"handle", &community.Handle}, {"icon", &community.Icon}, {"banner", &community.Banner}} {
		if value, ok := body[field.Key]; ok {
			*field.Value = strings.TrimSpace(Str(value))
		}
	}
	if community.Name == "" {
		Fail(400, "name is required")
	}
	if community.Handle == "" {
		community.Handle = strings.Trim(SlugRE.ReplaceAllString(strings.ToLower(community.Name), "-"), "-")
		if community.Handle == "" {
			community.Handle = "community-" + Token()[:8]
		}
	}
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&community).Error)
		if creating {
			return tx.Create(&models.CommunityMembership{CommunityID: community.ID, UserID: req.User.ID, Role: "admin"}).Error
		}
		return nil
	})
	if creating {
		req.Status = 201
	}
	return handler.CommunityJSON(community, req.User)
}
func (handler *Handler) CommunityJoin(req *Request) any {
	community := handler.Community(req)
	if strings.HasSuffix(req.R.URL.Path, "/leave") {
		if community.CreatedByID == req.User.ID {
			Fail(400, "Owner cannot leave the community")
		}
		Check(handler.DB.Where("community_id = ? AND user_id = ?", community.ID, req.User.ID).Delete(&models.CommunityMembership{}).Error)
	} else {
		Check(handler.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CommunityMembership{CommunityID: community.ID, UserID: req.User.ID, Role: "member"}).Error)
	}
	return handler.CommunityJSON(community, req.User)
}
