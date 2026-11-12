package links

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) LinkJSON(l models.Link) any {
	o := Object(l)
	tags := []string{}
	Check(s.DB.Model(&models.Hashtag{}).Where("id IN (?)", s.DB.Model(&models.LinkTag{}).Select("hashtag_id").Where("link_id = ?", l.ID)).Order("name").Pluck("name", &tags).Error)
	o["hashtags"] = tags
	return o
}
