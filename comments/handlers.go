package comments

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/tree"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Comments(req *Request) any {
	db := handler.DB.Model(&models.Comment{})
	if req.Param("id") != "" {
		id := req.ID("id")
		if strings.HasPrefix(req.R.URL.Path, "/videos/") {
			handler.VideoAccess(req, id)
			db = db.Where("id IN (?)", handler.DB.Model(&models.VideoComment{}).Select("comment_id").Where("video_id = ?", id))
		} else {
			Get[models.Post](handler.DB, id)
			db = db.Where("id IN (?)", handler.DB.Model(&models.PostComment{}).Select("comment_id").Where("post_id = ?", id))
		}
	}
	if author := req.R.URL.Query().Get("author"); author != "" {
		db = db.Where("user_id IN (?)", handler.DB.Model(&models.User{}).Select("id").Where("username = ?", author))
	}
	if friendly := req.R.URL.Query().Get("friendly_token"); friendly != "" {
		db = db.Where("friendly_token = ?", friendly)
	}
	if req.Param("comment") != "" {
		comment := Get[models.Comment](db, req.ID("comment"))
		return handler.CommentJSON(comment)
	}
	if req.Param("uid") != "" {
		var comment models.Comment
		Check(db.First(&comment, "uid = ?", req.Param("uid")).Error)
		if req.R.Method == "DELETE" {
			if !req.User.IsStaff {
				Owned(comment.UserID, req.User.ID)
			}
			req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "comments_comment", comment.ID) })
			req.Status = 204
			return nil
		}
		return handler.CommentJSON(comment)
	}
	if req.Param("id") != "" {
		out := []any{}
		for _, comment := range All[models.Comment](db.Order("add_date, id")) {
			out = append(out, handler.CommentJSON(comment))
		}
		return map[string]any{"comments": out}
	}
	return req.Page(db.Order("add_date DESC"), &[]models.Comment{}, func(value any) any {
		out := []any{}
		for _, comment := range *value.(*[]models.Comment) {
			out = append(out, handler.CommentJSON(comment))
		}
		return out
	})
}
func (handler *Handler) CommentCreate(req *Request) any {
	body := req.Body()
	text := strings.TrimSpace(Clean(Str(body["text"])))
	if text == "" || len([]rune(text)) > 10000 {
		Fail(400, "text must contain 1–10000 characters")
	}
	uuid := Token()[:32]
	uuid = uuid[:8] + "-" + uuid[8:12] + "-4" + uuid[13:16] + "-a" + uuid[17:20] + "-" + uuid[20:]
	comment := models.Comment{Text: text, UserID: req.User.ID, UID: uuid, FriendlyToken: Token()[:26]}
	if friendly := Str(body["friendly_token"]); friendly != "" {
		comment.FriendlyToken = friendly
	}
	video := strings.HasPrefix(req.R.URL.Path, "/videos/")
	if req.Param("id") != "" {
		if video {
			handler.VideoAccess(req, req.ID("id"))
		} else {
			Get[models.Post](handler.DB, req.ID("id"))
		}
	}
	if pid := body["parent_comment_id"]; pid != nil {
		parent := Get[models.Comment](handler.DB, Num(pid))
		comment.ParentID = Ptr(parent.ID)
	} else if uid := Str(body["parent"]); uid != "" {
		var parent models.Comment
		Check(handler.DB.First(&parent, "uid = ?", uid).Error)
		comment.ParentID = Ptr(parent.ID)
	}
	req.Tx(func(tx *gorm.DB) error {
		if comment.ParentID != nil && req.Param("id") != "" {
			var count int64
			query := tx.Model(&models.PostComment{}).Where("post_id = ? AND comment_id = ?", req.ID("id"), *comment.ParentID)
			if video {
				query = tx.Model(&models.VideoComment{}).Where("video_id = ? AND comment_id = ?", req.ID("id"), *comment.ParentID)
			}
			Check(query.Count(&count).Error)
			if count == 0 {
				Fail(400, "Parent comment belongs to another resource")
			}
		}
		Check(tx.Create(&comment).Error)
		Check(tree.Rebuild(tx, "comments_comment"))
		if req.Param("id") != "" {
			if video {
				return tx.Create(&models.VideoComment{VideoID: req.ID("id"), CommentID: comment.ID}).Error
			}
			return tx.Create(&models.PostComment{PostID: req.ID("id"), CommentID: comment.ID}).Error
		}
		return nil
	})
	req.Status = 201
	return handler.CommentJSON(comment)
}
