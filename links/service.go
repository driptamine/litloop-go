package links

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) SaveLink(tx *gorm.DB, userID int64, body map[string]any) models.Link {
	uri := Str(body["url"])
	if !ValidURL(uri) {
		Fail(400, "A valid HTTP URL is required")
	}
	link := models.Link{UserID: Ptr(userID), URL: uri, Title: Str(body["title"]), Status: "unread"}
	if value := Str(body["status"]); value != "" {
		link.Status = value
	}
	if value := Str(body["time_added"]); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			link.TimeAdded = &parsed
		} else if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
			stamp := time.Unix(unix, 0).UTC()
			link.TimeAdded = &stamp
		}
	}
	Check(tx.Create(&link).Error)
	tags, ok := body["tags"].([]any)
	if !ok && body["tags"] != nil {
		Fail(400, "tags must be an array")
	}
	for _, tag := range tags {
		name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(Str(tag))), "#")
		if name == "" {
			continue
		}
		if len([]rune(name)) > 100 {
			Fail(400, "Tag too long")
		}
		hashtag := models.Hashtag{Name: name}
		Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&hashtag).Error)
		Check(tx.First(&hashtag, "name = ?", name).Error)
		Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.LinkTag{LinkID: link.ID, HashtagID: hashtag.ID}).Error)
	}
	return link
}
