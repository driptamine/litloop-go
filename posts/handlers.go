package posts

import (
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) PostWrite(req *Request) any {
	var post models.Post
	if req.Param("id") != "" {
		post = Get[models.Post](handler.DB, req.ID("id"))
		Owned(Val(post.AuthorID), req.User.ID)
	} else {
		post = models.Post{AuthorID: Ptr(req.User.ID), FriendlyID: Token()[:12]}
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "posts_post", post.ID) })
		req.Status = 204
		return nil
	}
	body := req.Body()
	if value, ok := body["title"]; ok {
		post.Title = Truncate(Clean(Str(value)), 99)
	}
	if value, ok := body["description"]; ok {
		post.Description = Clean(Str(value))
	}
	if len([]rune(post.Description)) > 100 {
		Fail(400, "description exceeds 100 characters")
	}
	creating := post.ID == 0
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&post).Error)
		handler.PostFiles(req, tx, post, body)
		handler.LinkPost(tx, post, body, req.User.ID)
		return nil
	})
	if creating {
		req.Status = 201
	}
	out := handler.PostJSON(post, req.User)
	if creating {
		out["post_id"] = post.ID
		out["success"] = true
		out["message"] = "Post created successfully"
		for _, key := range []string{"photo_ids", "video_ids", "track_ids"} {
			if value, ok := body[key]; ok {
				out[key] = value
			}
		}
		if strings.HasSuffix(req.R.URL.Path, "/create_post_with_line_breaks") {
			out["content"] = strings.ReplaceAll(Str(body["description"]), "\n", "<br/>")
		}
	}
	return out
}
func (handler *Handler) Posts(req *Request) any {
	db := handler.DB.Model(&models.Post{})
	if req.Param("username") != "" {
		var user models.User
		Check(handler.DB.First(&user, "username = ?", req.Param("username")).Error)
		db = db.Where("author_id = ?", user.ID)
	} else if strings.HasSuffix(req.R.URL.Path, "/me") {
		db = db.Where("author_id = ?", req.User.ID)
	}
	var total int64
	Check(db.Count(&total).Error)
	db = db.Order("created_at DESC, id DESC")
	page, size := 1, 20
	feed := strings.HasSuffix(req.R.URL.Path, "/feed")
	if feed {
		page, _ = strconv.Atoi(req.R.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		if value, err := strconv.Atoi(req.R.URL.Query().Get("page_size")); err == nil && value > 0 {
			size = value
		}
		if size > 100 {
			size = 100
		}
		db = db.Offset((page - 1) * size).Limit(size)
	}
	items := []any{}
	for _, post := range All[models.Post](db) {
		items = append(items, handler.PostJSON(post, req.User))
	}
	if !feed && !strings.HasSuffix(req.R.URL.Path, "/list") && !strings.HasSuffix(req.R.URL.Path, "/list_no_drf") {
		return items
	}
	result := map[string]any{"posts": items}
	if feed {
		var next any
		if int64(page*size) < total {
			next = page + 1
		}
		result["total"] = total
		result["page"] = page
		result["page_size"] = size
		result["has_next"] = next != nil
		result["next_page"] = next
	}
	return result
}
func (handler *Handler) PostLike(req *Request) any {
	id := req.ID("id")
	var liked bool
	var count int64
	req.Tx(func(tx *gorm.DB) error {
		var post models.Post
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, id).Error)
		var like models.PostLike
		err := tx.First(&like, "post_id = ? AND user_id = ?", id, req.User.ID).Error
		if err == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.PostLike{PostID: id, UserID: req.User.ID}).Error)
			liked = true
		} else {
			Check(err)
			Check(tx.Delete(&like).Error)
		}
		Check(tx.Model(&models.PostLike{}).Where("post_id = ?", id).Count(&count).Error)
		Check(tx.Model(&post).Update("likes_count", count).Error)
		return engagement.EnqueueReaction(tx, handler.Config, "post", id, req.User.ID, liked)
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
