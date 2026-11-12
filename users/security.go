package users

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

func HashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 72 {
		return "", errors.New("password must contain 8–72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
func VerifyPassword(hash, password string) bool {
	if len(password) > 4096 {
		return false
	}
	if strings.HasPrefix(hash, "pbkdf2_sha256$") {
		p := strings.Split(hash, "$")
		if len(p) != 4 {
			return false
		}
		n, e := strconv.Atoi(p[1])
		if e != nil || n < 1 || n > 10000000 {
			return false
		}
		key := pbkdf2.Key([]byte(password), []byte(p[2]), n, 32, sha256.New)
		return subtle.ConstantTimeCompare([]byte(base64.StdEncoding.EncodeToString(key)), []byte(p[3])) == 1
	}
	if strings.HasPrefix(hash, "bcrypt_sha256$") {
		passwordBytes := sha256.Sum256([]byte(password))
		return bcrypt.CompareHashAndPassword([]byte(strings.TrimPrefix(hash, "bcrypt_sha256$")), []byte(hex.EncodeToString(passwordBytes[:]))) == nil
	}
	if strings.HasPrefix(hash, "bcrypt$") {
		hash = strings.TrimPrefix(hash, "bcrypt$")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
