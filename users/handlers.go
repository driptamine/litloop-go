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

func (handler *Handler) Signup(req *Request) any {
	body := req.Body()
	email := strings.ToLower(strings.TrimSpace(Str(body["email"])))
	if _, err := mail.ParseAddress(email); err != nil {
		Fail(400, "Valid email is required")
	}
	password := hashPassword(Str(body["password"]))
	username := strings.TrimSpace(Str(body["username"]))
	if username == "" {
		username = strings.ReplaceAll(strings.Split(email, "@")[0], ".", "_")
	}
	if len(username) > 255 {
		Fail(400, "Username too long")
	}
	var count int64
	Check(handler.DB.Model(&models.User{}).Where("username = ?", username).Count(&count).Error)
	if count > 0 || handler.usernameReserved(username) {
		username += "_" + Token()[:8]
	}
	avatar := Str(body["avatar"])
	if avatar != "" && handler.Config.UsesR2ForMedia() {
		Fail(400, "Upload the avatar after creating an account")
	}
	user := models.User{Email: email, Username: username, Password: password, Avatar: avatar, IsActive: true}
	req.Tx(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		chat := models.Chat{ChatType: "direct", Name: fmt.Sprintf("saved-%d", user.ID), IsSavedMessages: true}
		if err := tx.Create(&chat).Error; err != nil {
			return err
		}
		return tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: user.ID}).Error
	})
	req.Status = 201
	return handler.Tokens(user)
}
func (handler *Handler) Signin(req *Request) any {
	body := req.Body()
	login := Str(body["email_or_username"])
	if login == "" {
		login = Str(body["email"])
	}
	var user models.User
	if handler.DB.Where(
		"(LOWER(email) = ? OR username = ?) AND is_active = ?",
		strings.ToLower(login),
		login,
		true,
	).First(&user).Error != nil || !verifyPassword(user.Password, Str(body["password"])) {
		Fail(401, "Invalid credentials")
	}
	Check(handler.DB.Model(&user).Update("last_login", time.Now()).Error)
	return handler.Tokens(user)
}
func (handler *Handler) Refresh(req *Request) any {
	claims := handler.Claims(Str(req.Body()["refresh"]), "refresh")
	if claims == nil {
		Fail(401, "Invalid refresh token")
	}
	user := Get[models.User](handler.DB, claims["user_id"])
	if !user.IsActive {
		Fail(401, "Inactive account")
	}
	return handler.Tokens(user)
}
func (handler *Handler) UserDetail(req *Request) any {
	var user models.User
	key := req.Param("key")
	switch key {
	case "signup", "signin", "callback", "token", "password-reset":
		Fail(405, "Method not allowed")
	}
	id, err := strconv.ParseInt(key, 10, 64)
	if err == nil {
		Check(handler.DB.First(&user, "id = ?", id).Error)
	} else {
		Check(handler.DB.First(&user, "username = ?", key).Error)
	}
	return user
}
func (handler *Handler) UserList(req *Request) any {
	Search := strings.HasSuffix(req.R.URL.Path, "/search")
	term := strings.TrimSpace(req.R.URL.Query().Get("q"))
	if Search && term == "" {
		return map[string]any{"results": []any{}}
	}
	db := handler.DB.Model(&models.User{}).Order("id")
	if Search {
		like := "%" + strings.ToLower(term) + "%"
		db = db.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", like, like).Limit(50)
	}
	out := []any{}
	for _, user := range All[models.User](db) {
		item := Object(PublicUser(user))
		if Search {
			item["is_verified"] = user.IsVerified
		} else {
			item["is_staff"] = user.IsStaff
			item["is_superuser"] = user.IsSuperuser
		}
		out = append(out, item)
	}
	result := map[string]any{"count": len(out), "results": out}
	if Search {
		result["query"] = term
	}
	return result
}
func (handler *Handler) UpdateProfile(req *Request) any {
	body := req.Body()
	updates := map[string]any{}
	if value, ok := body["username"]; ok {
		username := strings.TrimSpace(Str(value))
		if username == "" {
			Fail(400, "Username cannot be empty")
		}
		if len(username) > 255 {
			Fail(400, "Username too long")
		}
		var count int64
		Check(handler.DB.Model(&models.User{}).Where("username = ? AND id <> ?", username, req.User.ID).Count(&count).Error)
		if count > 0 {
			Fail(409, "Username is taken")
		}
		if handler.reservedUsernameTaken(username, req.User.ID) {
			Fail(409, "Username is reserved")
		}
		updates["username"] = username
	}
	if value, ok := body["email"]; ok {
		email := strings.ToLower(strings.TrimSpace(Str(value)))
		if _, err := mail.ParseAddress(email); err != nil {
			Fail(400, "Valid email is required")
		}
		var count int64
		Check(handler.DB.Model(&models.User{}).Where("email = ? AND id <> ?", email, req.User.ID).Count(&count).Error)
		if count > 0 {
			Fail(409, "Email is taken")
		}
		updates["email"] = email
	}
	if len(updates) == 0 {
		Fail(400, "Nothing to update")
	}
	Check(handler.DB.Model(&models.User{ID: req.User.ID}).Updates(updates).Error)
	if name, ok := updates["username"]; ok {
		Check(handler.DB.Where("username = ? AND assigned_to_id = ?", Str(name), req.User.ID).Delete(&models.ReservedUsername{}).Error)
	}
	return PublicUser(Get[models.User](handler.DB, req.User.ID))
}

func (handler *Handler) ChangePassword(req *Request) any {
	body := req.Body()
	if !verifyPassword(req.User.Password, Str(body["old_password"])) {
		Fail(401, "Current password is incorrect")
	}
	password := hashPassword(Str(body["new_password"]))
	Check(handler.DB.Model(&models.User{ID: req.User.ID}).Update("password", password).Error)
	return map[string]any{"status": "ok"}
}

func (handler *Handler) Role(req *Request) any {
	req.Admin()
	user := Get[models.User](handler.DB, req.ID("id"))
	body := req.Body()
	updates := map[string]any{}
	for _, key := range []string{"is_staff", "is_superuser"} {
		if value, ok := body[key]; ok {
			updates[key] = Boolean(value)
		}
	}
	Check(handler.DB.Model(&user).Updates(updates).Error)
	return Get[models.User](handler.DB, user.ID)
}
func (handler *Handler) Logout(req *Request) any {
	raw := strings.TrimPrefix(req.R.Header.Get("Authorization"), "Bearer ")
	if claims := handler.Claims(raw, "access"); claims != nil && Str(claims["jti"]) != "" {
		exp, _ := claims.GetExpirationTime()
		Check(handler.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.RevokedToken{ID: Str(claims["jti"]), ExpiresAt: exp.Time}).Error)
	}
	authcookie.ClearAccessToken(req.W)
	return map[string]any{"message": "Logged out"}
}
