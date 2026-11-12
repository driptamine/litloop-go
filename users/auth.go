package users

import (
	"fmt"
	authcookie "litloop/backend/auth_cookie"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	JWT "github.com/golang-jwt/jwt/v5"
)

func hashPassword(p string) string {
	hash, err := HashPassword(p)
	if err != nil {
		Fail(400, err.Error())
	}
	return hash
}
func verifyPassword(hash, password string) bool { return VerifyPassword(hash, password) }
func (s *Handler) JWT(user int64, kind string, ttl time.Duration) string {
	now := time.Now()
	t := JWT.NewWithClaims(JWT.SigningMethodHS256, JWT.MapClaims{"user_id": user, "token_type": kind, "iat": now.Unix(), "exp": now.Add(ttl).Unix(), "jti": Token()})
	v, e := t.SignedString([]byte(s.Config.JWTSecret))
	Check(e)
	return v
}
func (s *Handler) Claims(raw string, kind string) JWT.MapClaims {
	t, e := JWT.Parse(raw, func(t *JWT.Token) (any, error) { return []byte(s.Config.JWTSecret), nil }, JWT.WithValidMethods([]string{"HS256"}), JWT.WithExpirationRequired())
	if e != nil || !t.Valid {
		return nil
	}
	c, ok := t.Claims.(JWT.MapClaims)
	if !ok {
		return nil
	}
	if typ := Str(c["token_type"]); typ != "" && typ != kind {
		return nil
	}
	if kind == "refresh" && Str(c["token_type"]) != "refresh" {
		return nil
	}
	var n int64
	if jti := Str(c["jti"]); jti != "" {
		if s.DB.Model(&models.RevokedToken{}).Where("id = ?", jti).Count(&n).Error != nil || n > 0 {
			return nil
		}
	}
	return c
}
func (s *Handler) Authenticate(r *http.Request) *models.User {
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if raw == "" {
		raw = authcookie.AccessToken(r)
	}
	if raw == "" && strings.HasPrefix(r.URL.Path, "/ws/") {
		raw = r.URL.Query().Get("token")
	}
	if raw == "" {
		return nil
	}
	Claims := s.Claims(raw, "access")
	if Claims == nil {
		return nil
	}
	id, e := strconv.ParseInt(fmt.Sprint(Claims["user_id"]), 10, 64)
	if e != nil {
		return nil
	}
	var u models.User
	if s.DB.Where("id = ? AND is_active = ?", id, true).First(&u).Error != nil {
		return nil
	}
	return &u
}
func (s *Handler) Tokens(u models.User) map[string]any {
	access := s.JWT(u.ID, "access", s.Config.AccessTTL)
	return map[string]any{"user": PublicUser(u), "token": access, "access": access, "refresh": s.JWT(u.ID, "refresh", s.Config.RefreshTTL)}
}
