package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) CommunityPosts(q *Request) any {
	c := s.Community(q)
	path := q.R.URL.Path
	if strings.HasSuffix(path, "/request") {
		b := q.Body()
		var membership models.CommunityMembership
		Check(s.DB.First(&membership, "community_id = ? AND user_id = ?", c.ID, q.User.ID).Error)
		var cp models.CommunityPost
		q.Tx(func(tx *gorm.DB) error {
			var p models.Post
			if id, ok := b["post_id"]; ok {
				p = Get[models.Post](tx, Num(id))
				Owned(Val(p.AuthorID), q.User.ID)
			} else {
				title, description := strings.TrimSpace(Clean(Str(b["title"]))), strings.TrimSpace(Clean(Str(b["description"])))
				if title == "" && description == "" && len(IDs(b["photo_ids"]))+len(IDs(b["video_ids"]))+len(IDs(b["track_ids"]))+len(IDs(b["playlist_ids"])) == 0 {
					Fail(400, "title, description, or media is required")
				}
				if len([]rune(description)) > 100 {
					Fail(400, "description exceeds 100 characters")
				}
				p = models.Post{Title: Truncate(title, 99), Description: description, AuthorID: Ptr(q.User.ID), FriendlyID: Token()[:12]}
				Check(tx.Create(&p).Error)
				s.LinkPost(tx, p, b, q.User.ID)
			}
			cp = models.CommunityPost{CommunityID: c.ID, PostID: p.ID, RequestedByID: q.User.ID, Status: "pending"}
			if membership.Role == "admin" || membership.Role == "moderator" {
				cp.Status = "approved"
				cp.ReviewedByID = Ptr(q.User.ID)
				cp.ReviewedAt = Ptr(time.Now())
			}
			return tx.Create(&cp).Error
		})
		q.Status = 201
		o := s.CommunityPostJSON(cp, q.User)

		o["post_id"] = cp.PostID
		if _, ok := b["post_id"]; ok {
			o["id"] = cp.ID
		}
		return o
	}
	if q.Param("cp") != "" {
		cp := Get[models.CommunityPost](s.DB.Where("community_id = ?", c.ID), q.ID("cp"))
		if q.R.Method == "DELETE" {
			if !(cp.RequestedByID == q.User.ID && cp.Status == "pending") {
				s.CommunityRole(c, q.User, true)
			}
			Check(s.DB.Delete(&cp).Error)
			return map[string]any{"deleted": true, "post_id": cp.PostID}
		}
		s.CommunityRole(c, q.User, true)
		q.Tx(func(tx *gorm.DB) error {
			Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cp, cp.ID).Error)
			if cp.Status != "pending" {
				Fail(400, "Post is already "+cp.Status)
			}
			status := "approved"
			if strings.HasSuffix(path, "/reject") {
				status = "rejected"
			}
			return tx.Model(&cp).Updates(map[string]any{"status": status, "reviewed_by_id": q.User.ID, "reviewed_at": time.Now()}).Error
		})
		return s.CommunityPostJSON(Get[models.CommunityPost](s.DB, cp.ID), q.User)
	}
	status := "approved"
	if strings.HasSuffix(path, "/pending") {
		s.CommunityRole(c, q.User, true)
		status = "pending"
	}
	out := []any{}
	for _, cp := range All[models.CommunityPost](s.DB.Where("community_id = ? AND status = ?", c.ID, status).Order("created_at DESC")) {
		out = append(out, s.CommunityPostJSON(cp, q.User))
	}
	return map[string]any{"posts": out}
}
