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

func (s *Handler) SaveLink(tx *gorm.DB, u int64, b map[string]any) models.Link {
	uri := Str(b["url"])
	if !ValidURL(uri) {
		Fail(400, "A valid HTTP URL is required")
	}
	l := models.Link{UserID: Ptr(u), URL: uri, Title: Str(b["title"]), Status: "unread"}
	if v := Str(b["status"]); v != "" {
		l.Status = v
	}
	if v := Str(b["time_added"]); v != "" {
		t, e := time.Parse(time.RFC3339, v)
		if e == nil {
			l.TimeAdded = &t
		} else if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
			stamp := time.Unix(unix, 0).UTC()
			l.TimeAdded = &stamp
		}
	}
	Check(tx.Create(&l).Error)
	tags, ok := b["tags"].([]any)
	if !ok && b["tags"] != nil {
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
		h := models.Hashtag{Name: name}
		Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&h).Error)
		Check(tx.First(&h, "name = ?", name).Error)
		Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.LinkTag{LinkID: l.ID, HashtagID: h.ID}).Error)
	}
	return l
}
