package httpapi

import (
	"crypto/hmac"
	"litloop/backend/internal/models"
)

func PublicUser(user models.User) any {
	return map[string]any{"id": user.ID, "username": user.Username, "email": user.Email, "avatar": user.Avatar}
}
func SecureEqual(actual, expected string) bool	{ return hmac.Equal([]byte(actual), []byte(expected)) }
