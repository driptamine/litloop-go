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

func hashPassword(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		Fail(400, err.Error())
	}
	return hash
}
func verifyPassword(hash, password string) bool	{ return VerifyPassword(hash, password) }
func (handler *Handler) JWT(user int64, kind string, ttl time.Duration) string {
	now := time.Now()
	jwtToken := JWT.NewWithClaims(JWT.SigningMethodHS256, JWT.MapClaims{"user_id": user, "token_type": kind, "iat": now.Unix(), "exp": now.Add(ttl).Unix(), "jti": Token()})
	token, err := jwtToken.SignedString([]byte(handler.Config.JWTSecret))
	Check(err)
	return token
}
func (handler *Handler) Claims(raw string, kind string) JWT.MapClaims {
	parsedToken, err := JWT.Parse(raw, func(parsedToken *JWT.Token) (any, error) { return []byte(handler.Config.JWTSecret), nil }, JWT.WithValidMethods([]string{"HS256"}), JWT.WithExpirationRequired())
	if err != nil || !parsedToken.Valid {
		return nil
	}
	claims, ok := parsedToken.Claims.(JWT.MapClaims)
	if !ok {
		return nil
	}
	if typ := Str(claims["token_type"]); typ != "" && typ != kind {
		return nil
	}
	if kind == "refresh" && Str(claims["token_type"]) != "refresh" {
		return nil
	}
	var count int64
	if jti := Str(claims["jti"]); jti != "" {
		if handler.DB.Model(&models.RevokedToken{}).Where("id = ?", jti).Count(&count).Error != nil || count > 0 {
			return nil
		}
	}
	return claims
}
func (handler *Handler) Authenticate(httpReq *http.Request) *models.User {
	raw := strings.TrimPrefix(httpReq.Header.Get("Authorization"), "Bearer ")
	if raw == "" {
		raw = authcookie.AccessToken(httpReq)
	}
	if raw == "" && strings.HasPrefix(httpReq.URL.Path, "/ws/") {
		raw = httpReq.URL.Query().Get("token")
	}
	if raw == "" {
		return nil
	}
	Claims := handler.Claims(raw, "access")
	if Claims == nil {
		return nil
	}
	id, err := strconv.ParseInt(fmt.Sprint(Claims["user_id"]), 10, 64)
	if err != nil {
		return nil
	}
	var user models.User
	if handler.DB.Where("id = ? AND is_active = ?", id, true).First(&user).Error != nil {
		return nil
	}
	return &user
}
func (handler *Handler) Tokens(user models.User) map[string]any {
	access := handler.JWT(user.ID, "access", handler.Config.AccessTTL)
	return map[string]any{"user": PublicUser(user), "token": access, "access": access, "refresh": handler.JWT(user.ID, "refresh", handler.Config.RefreshTTL)}
}
