package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) PostWrite(q *Request) any {
	var p models.Post
	if q.Param("id") != "" {
		p = Get[models.Post](s.DB, q.ID("id"))
		Owned(Val(p.AuthorID), q.User.ID)
	} else {
		p = models.Post{AuthorID: Ptr(q.User.ID), FriendlyID: Token()[:12]}
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "posts_post", p.ID) })
		q.Status = 204
		return nil
	}
	b := q.Body()
	if v, ok := b["title"]; ok {
		p.Title = Truncate(Clean(Str(v)), 99)
	}
	if v, ok := b["description"]; ok {
		p.Description = Clean(Str(v))
	}
	if len([]rune(p.Description)) > 100 {
		Fail(400, "description exceeds 100 characters")
	}
	creating := p.ID == 0
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&p).Error)
		s.PostFiles(q, tx, p, b)
		s.LinkPost(tx, p, b, q.User.ID)
		return nil
	})
	if creating {
		q.Status = 201
	}
	o := s.PostJSON(p, q.User)
	if creating {
		o["post_id"] = p.ID
		o["success"] = true
		o["message"] = "Post created successfully"
		for _, key := range []string{"photo_ids", "video_ids", "track_ids"} {
			if v, ok := b[key]; ok {
				o[key] = v
			}
		}
		if strings.HasSuffix(q.R.URL.Path, "/create_post_with_line_breaks") {
			o["content"] = strings.ReplaceAll(Str(b["description"]), "\n", "<br/>")
		}
	}
	return o
}
func (s *Handler) Posts(q *Request) any {
	db := s.DB.Model(&models.Post{})
	if q.Param("username") != "" {
		var u models.User
		Check(s.DB.First(&u, "username = ?", q.Param("username")).Error)
		db = db.Where("author_id = ?", u.ID)
	} else if strings.HasSuffix(q.R.URL.Path, "/me") {
		db = db.Where("author_id = ?", q.User.ID)
	}
	var total int64
	Check(db.Count(&total).Error)
	db = db.Order("created_at DESC, id DESC")
	page, size := 1, 20
	feed := strings.HasSuffix(q.R.URL.Path, "/feed")
	if feed {
		page, _ = strconv.Atoi(q.R.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		if v, e := strconv.Atoi(q.R.URL.Query().Get("page_size")); e == nil && v > 0 {
			size = v
		}
		if size > 100 {
			size = 100
		}
		db = db.Offset((page - 1) * size).Limit(size)
	}
	items := []any{}
	for _, p := range All[models.Post](db) {
		items = append(items, s.PostJSON(p, q.User))
	}
	if !feed && !strings.HasSuffix(q.R.URL.Path, "/list") && !strings.HasSuffix(q.R.URL.Path, "/list_no_drf") {
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
func (s *Handler) PostLike(q *Request) any {
	id := q.ID("id")
	var liked bool
	var count int64
	q.Tx(func(tx *gorm.DB) error {
		var p models.Post
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error)
		var like models.PostLike
		e := tx.First(&like, "post_id = ? AND user_id = ?", id, q.User.ID).Error
		if e == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.PostLike{PostID: id, UserID: q.User.ID}).Error)
			liked = true
		} else {
			Check(e)
			Check(tx.Delete(&like).Error)
		}
		Check(tx.Model(&models.PostLike{}).Where("post_id = ?", id).Count(&count).Error)
		return tx.Model(&p).Update("likes_count", count).Error
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
