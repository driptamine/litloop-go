package httpapi

import (
	"crypto/hmac"
	"litloop/backend/internal/models"
)

func PublicUser(u models.User) any {
	return map[string]any{"id": u.ID, "username": u.Username, "email": u.Email, "avatar": u.Avatar}
}
func SecureEqual(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }
