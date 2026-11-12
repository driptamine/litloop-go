package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) Communities(q *Request) any {
	out := []any{}
	db := s.DB.Order("id DESC")
	if Search := q.R.URL.Query().Get("q"); Search != "" {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(Search)+"%")
	}
	for _, c := range All[models.Community](db) {
		out = append(out, s.CommunityJSON(c, q.User))
	}
	return map[string]any{"communities": out}
}
func (s *Handler) CommunityWrite(q *Request) any {
	var c models.Community
	creating := q.Param("id") == ""
	if !creating {
		c = s.Community(q)
		s.CommunityRole(c, q.User, false)
	} else {
		c.CreatedByID = q.User.ID
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "communities_community", c.ID) })
		return map[string]any{"deleted": true, "community_id": c.ID}
	}
	b := q.Body()
	for _, p := range []struct {
		Key   string
		Value *string
	}{{"name", &c.Name}, {"description", &c.Description}, {"handle", &c.Handle}, {"icon", &c.Icon}, {"banner", &c.Banner}} {
		if v, ok := b[p.Key]; ok {
			*p.Value = strings.TrimSpace(Str(v))
		}
	}
	if c.Name == "" {
		Fail(400, "name is required")
	}
	if c.Handle == "" {
		c.Handle = strings.Trim(SlugRE.ReplaceAllString(strings.ToLower(c.Name), "-"), "-")
		if c.Handle == "" {
			c.Handle = "community-" + Token()[:8]
		}
	}
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&c).Error)
		if creating {
			return tx.Create(&models.CommunityMembership{CommunityID: c.ID, UserID: q.User.ID, Role: "admin"}).Error
		}
		return nil
	})
	if creating {
		q.Status = 201
	}
	return s.CommunityJSON(c, q.User)
}
func (s *Handler) CommunityJoin(q *Request) any {
	c := s.Community(q)
	if strings.HasSuffix(q.R.URL.Path, "/leave") {
		if c.CreatedByID == q.User.ID {
			Fail(400, "Owner cannot leave the community")
		}
		Check(s.DB.Where("community_id = ? AND user_id = ?", c.ID, q.User.ID).Delete(&models.CommunityMembership{}).Error)
	} else {
		Check(s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CommunityMembership{CommunityID: c.ID, UserID: q.User.ID, Role: "member"}).Error)
	}
	return s.CommunityJSON(c, q.User)
}
