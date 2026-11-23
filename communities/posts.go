package communities

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) CommunityPosts(req *Request) any {
	community := handler.Community(req)
	path := req.R.URL.Path
	if strings.HasSuffix(path, "/request") {
		body := req.Body()
		var membership models.CommunityMembership
		Check(handler.DB.First(&membership, "community_id = ? AND user_id = ?", community.ID, req.User.ID).Error)
		var cp models.CommunityPost
		req.Tx(func(tx *gorm.DB) error {
			var post models.Post
			if id, ok := body["post_id"]; ok {
				post = Get[models.Post](tx, Num(id))
				Owned(Val(post.AuthorID), req.User.ID)
			} else {
				title, description := strings.TrimSpace(Clean(Str(body["title"]))), strings.TrimSpace(Clean(Str(body["description"])))
				if title == "" && description == "" && len(IDs(body["photo_ids"]))+len(IDs(body["video_ids"]))+len(IDs(body["track_ids"]))+len(IDs(body["playlist_ids"])) == 0 {
					Fail(400, "title, description, or media is required")
				}
				if len([]rune(description)) > 100 {
					Fail(400, "description exceeds 100 characters")
				}
				post = models.Post{Title: Truncate(title, 99), Description: description, AuthorID: Ptr(req.User.ID), FriendlyID: Token()[:12]}
				Check(tx.Create(&post).Error)
				handler.LinkPost(tx, post, body, req.User.ID)
			}
			cp = models.CommunityPost{CommunityID: community.ID, PostID: post.ID, RequestedByID: req.User.ID, Status: "pending"}
			if membership.Role == "admin" || membership.Role == "moderator" {
				cp.Status = "approved"
				cp.ReviewedByID = Ptr(req.User.ID)
				cp.ReviewedAt = Ptr(time.Now())
			}
			return tx.Create(&cp).Error
		})
		req.Status = 201
		postJSON := handler.CommunityPostJSON(cp, req.User)

		postJSON["post_id"] = cp.PostID
		if _, ok := body["post_id"]; ok {
			postJSON["id"] = cp.ID
		}
		return postJSON
	}
	if req.Param("cp") != "" {
		cp := Get[models.CommunityPost](handler.DB.Where("community_id = ?", community.ID), req.ID("cp"))
		if req.R.Method == "DELETE" {
			if !(cp.RequestedByID == req.User.ID && cp.Status == "pending") {
				handler.CommunityRole(community, req.User, true)
			}
			Check(handler.DB.Delete(&cp).Error)
			return map[string]any{"deleted": true, "post_id": cp.PostID}
		}
		handler.CommunityRole(community, req.User, true)
		req.Tx(func(tx *gorm.DB) error {
			Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cp, cp.ID).Error)
			if cp.Status != "pending" {
				Fail(400, "Post is already "+cp.Status)
			}
			status := "approved"
			if strings.HasSuffix(path, "/reject") {
				status = "rejected"
			}
			return tx.Model(&cp).Updates(map[string]any{"status": status, "reviewed_by_id": req.User.ID, "reviewed_at": time.Now()}).Error
		})
		return handler.CommunityPostJSON(Get[models.CommunityPost](handler.DB, cp.ID), req.User)
	}
	status := "approved"
	if strings.HasSuffix(path, "/pending") {
		handler.CommunityRole(community, req.User, true)
		status = "pending"
	}
	out := []any{}
	for _, cp := range All[models.CommunityPost](handler.DB.Where("community_id = ? AND status = ?", community.ID, status).Order("created_at DESC")) {
		out = append(out, handler.CommunityPostJSON(cp, req.User))
	}
	return map[string]any{"posts": out}
}
