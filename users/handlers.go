package users

import (
	"fmt"
	authcookie "litloop/backend/auth_cookie"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) Signup(q *Request) any {
	b := q.Body()
	email := strings.ToLower(strings.TrimSpace(Str(b["email"])))
	if _, e := mail.ParseAddress(email); e != nil {
		Fail(400, "Valid email is required")
	}
	password := hashPassword(Str(b["password"]))
	username := strings.TrimSpace(Str(b["username"]))
	if username == "" {
		username = strings.ReplaceAll(strings.Split(email, "@")[0], ".", "_")
	}
	if len(username) > 255 {
		Fail(400, "Username too long")
	}
	var n int64
	Check(s.DB.Model(&models.User{}).Where("username = ?", username).Count(&n).Error)
	if n > 0 {
		username += "_" + Token()[:8]
	}
	u := models.User{Email: email, Username: username, Password: password, Avatar: Str(b["avatar"]), IsActive: true}
	q.Tx(func(tx *gorm.DB) error {
		if e := tx.Create(&u).Error; e != nil {
			return e
		}
		chat := models.Chat{ChatType: "direct", Name: fmt.Sprintf("saved-%d", u.ID), IsSavedMessages: true}
		if e := tx.Create(&chat).Error; e != nil {
			return e
		}
		return tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: u.ID}).Error
	})
	q.Status = 201
	return s.Tokens(u)
}
func (s *Handler) Signin(q *Request) any {
	b := q.Body()
	login := Str(b["email_or_username"])
	if login == "" {
		login = Str(b["email"])
	}
	var u models.User
	if s.DB.Where(
		"(LOWER(email) = ? OR username = ?) AND is_active = ?",
		strings.ToLower(login),
		login,
		true,
	).First(&u).Error != nil || !verifyPassword(u.Password, Str(b["password"])) {
		Fail(401, "Invalid credentials")
	}
	Check(s.DB.Model(&u).Update("last_login", time.Now()).Error)
	return s.Tokens(u)
}
func (s *Handler) Refresh(q *Request) any {
	c := s.Claims(Str(q.Body()["refresh"]), "refresh")
	if c == nil {
		Fail(401, "Invalid refresh token")
	}
	u := Get[models.User](s.DB, c["user_id"])
	if !u.IsActive {
		Fail(401, "Inactive account")
	}
	return s.Tokens(u)
}
func (s *Handler) UserDetail(q *Request) any {
	var u models.User
	key := q.Param("key")
	switch key {
	case "signup", "signin", "callback", "token", "password-reset":
		Fail(405, "Method not allowed")
	}
	id, e := strconv.ParseInt(key, 10, 64)
	if e == nil {
		Check(s.DB.First(&u, "id = ?", id).Error)
	} else {
		Check(s.DB.First(&u, "username = ?", key).Error)
	}
	return u
}
func (s *Handler) UserList(q *Request) any {
	Search := strings.HasSuffix(q.R.URL.Path, "/search")
	term := strings.TrimSpace(q.R.URL.Query().Get("q"))
	if Search && term == "" {
		return map[string]any{"results": []any{}}
	}
	db := s.DB.Model(&models.User{}).Order("id")
	if Search {
		like := "%" + strings.ToLower(term) + "%"
		db = db.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", like, like).Limit(50)
	}
	out := []any{}
	for _, u := range All[models.User](db) {
		item := Object(PublicUser(u))
		if Search {
			item["is_verified"] = u.IsVerified
		} else {
			item["is_staff"] = u.IsStaff
			item["is_superuser"] = u.IsSuperuser
		}
		out = append(out, item)
	}
	result := map[string]any{"count": len(out), "results": out}
	if Search {
		result["query"] = term
	}
	return result
}
func (s *Handler) Role(q *Request) any {
	q.Admin()
	u := Get[models.User](s.DB, q.ID("id"))
	b := q.Body()
	updates := map[string]any{}
	for _, k := range []string{"is_staff", "is_superuser"} {
		if v, ok := b[k]; ok {
			updates[k] = Boolean(v)
		}
	}
	Check(s.DB.Model(&u).Updates(updates).Error)
	return Get[models.User](s.DB, u.ID)
}
func (s *Handler) Logout(q *Request) any {
	raw := strings.TrimPrefix(q.R.Header.Get("Authorization"), "Bearer ")
	if c := s.Claims(raw, "access"); c != nil && Str(c["jti"]) != "" {
		exp, _ := c.GetExpirationTime()
		Check(s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.RevokedToken{ID: Str(c["jti"]), ExpiresAt: exp.Time}).Error)
	}
	authcookie.ClearAccessToken(q.W)
	return map[string]any{"message": "Logged out"}
}
