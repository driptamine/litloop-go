package links

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) LinkJSON(link models.Link) any {
	out := Object(link)
	tags := []string{}
	Check(handler.DB.Model(&models.Hashtag{}).Where("id IN (?)", handler.DB.Model(&models.LinkTag{}).Select("hashtag_id").Where("link_id = ?", link.ID)).Order("name").Pluck("name", &tags).Error)
	out["hashtags"] = tags
	return out
}
