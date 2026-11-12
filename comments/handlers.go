package comments

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/tree"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Comments(q *Request) any {
	db := s.DB.Model(&models.Comment{})
	if q.Param("id") != "" {
		id := q.ID("id")
		if strings.HasPrefix(q.R.URL.Path, "/videos/") {
			s.VideoAccess(q, id)
			db = db.Where("id IN (?)", s.DB.Model(&models.VideoComment{}).Select("comment_id").Where("video_id = ?", id))
		} else {
			Get[models.Post](s.DB, id)
			db = db.Where("id IN (?)", s.DB.Model(&models.PostComment{}).Select("comment_id").Where("post_id = ?", id))
		}
	}
	if author := q.R.URL.Query().Get("author"); author != "" {
		db = db.Where("user_id IN (?)", s.DB.Model(&models.User{}).Select("id").Where("username = ?", author))
	}
	if friendly := q.R.URL.Query().Get("friendly_token"); friendly != "" {
		db = db.Where("friendly_token = ?", friendly)
	}
	if q.Param("comment") != "" {
		c := Get[models.Comment](db, q.ID("comment"))
		return s.CommentJSON(c)
	}
	if q.Param("uid") != "" {
		var c models.Comment
		Check(db.First(&c, "uid = ?", q.Param("uid")).Error)
		if q.R.Method == "DELETE" {
			if !q.User.IsStaff {
				Owned(c.UserID, q.User.ID)
			}
			q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "comments_comment", c.ID) })
			q.Status = 204
			return nil
		}
		return s.CommentJSON(c)
	}
	if q.Param("id") != "" {
		out := []any{}
		for _, c := range All[models.Comment](db.Order("add_date, id")) {
			out = append(out, s.CommentJSON(c))
		}
		return map[string]any{"comments": out}
	}
	return q.Page(db.Order("add_date DESC"), &[]models.Comment{}, func(v any) any {
		out := []any{}
		for _, c := range *v.(*[]models.Comment) {
			out = append(out, s.CommentJSON(c))
		}
		return out
	})
}
func (s *Handler) CommentCreate(q *Request) any {
	b := q.Body()
	text := strings.TrimSpace(Clean(Str(b["text"])))
	if text == "" || len([]rune(text)) > 10000 {
		Fail(400, "text must contain 1–10000 characters")
	}
	uuid := Token()[:32]
	uuid = uuid[:8] + "-" + uuid[8:12] + "-4" + uuid[13:16] + "-a" + uuid[17:20] + "-" + uuid[20:]
	c := models.Comment{Text: text, UserID: q.User.ID, UID: uuid, FriendlyToken: Token()[:26]}
	if friendly := Str(b["friendly_token"]); friendly != "" {
		c.FriendlyToken = friendly
	}
	video := strings.HasPrefix(q.R.URL.Path, "/videos/")
	if q.Param("id") != "" {
		if video {
			s.VideoAccess(q, q.ID("id"))
		} else {
			Get[models.Post](s.DB, q.ID("id"))
		}
	}
	if pid := b["parent_comment_id"]; pid != nil {
		parent := Get[models.Comment](s.DB, Num(pid))
		c.ParentID = Ptr(parent.ID)
	} else if uid := Str(b["parent"]); uid != "" {
		var parent models.Comment
		Check(s.DB.First(&parent, "uid = ?", uid).Error)
		c.ParentID = Ptr(parent.ID)
	}
	q.Tx(func(tx *gorm.DB) error {
		if c.ParentID != nil && q.Param("id") != "" {
			var count int64
			query := tx.Model(&models.PostComment{}).Where("post_id = ? AND comment_id = ?", q.ID("id"), *c.ParentID)
			if video {
				query = tx.Model(&models.VideoComment{}).Where("video_id = ? AND comment_id = ?", q.ID("id"), *c.ParentID)
			}
			Check(query.Count(&count).Error)
			if count == 0 {
				Fail(400, "Parent comment belongs to another resource")
			}
		}
		Check(tx.Create(&c).Error)
		Check(tree.Rebuild(tx, "comments_comment"))
		if q.Param("id") != "" {
			if video {
				return tx.Create(&models.VideoComment{VideoID: q.ID("id"), CommentID: c.ID}).Error
			}
			return tx.Create(&models.PostComment{PostID: q.ID("id"), CommentID: c.ID}).Error
		}
		return nil
	})
	q.Status = 201
	return s.CommentJSON(c)
}
