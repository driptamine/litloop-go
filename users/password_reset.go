package users

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) ForgotPassword(q *Request) any {
	email := strings.ToLower(Str(q.Body()["email"]))
	if email == "" {
		Fail(400, "Email is required")
	}
	if s.Config.SMTPAddr == "" {
		Fail(503, "SMTP is not configured")
	}
	var u models.User
	if s.DB.First(&u, "email = ?", email).Error == nil {
		raw := Token()
		sum := sha256.Sum256([]byte(raw))
		Check(s.DB.Create(&models.ResetToken{ID: hex.EncodeToString(sum[:]), UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}).Error)
		uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(u.ID, 10)))
		link := strings.TrimRight(s.Config.FrontendURL, "/") + "/password-reset/" + uid + "/" + raw
		Check(sendMail(s.Config.SMTPAddr, s.Config.SMTPUser, s.Config.SMTPPassword, s.Config.SMTPFrom, u.Email, "Password reset", link))
	}
	return map[string]any{"message": "If an account exists, a reset link has been sent."}
}
func sendMail(addr, user, password, from, to, subject, body string) error {
	host := strings.Split(addr, ":")[0]
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}
	for _, v := range []string{from, to, subject} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("invalid mail header")
		}
	}
	msg := []byte("From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
func (s *Handler) ResetPassword(q *Request) any {
	uid, e := base64.RawURLEncoding.DecodeString(q.Param("uid"))
	if e != nil {
		Fail(400, "Invalid reset link")
	}
	hash := sha256.Sum256([]byte(q.Param("token")))
	password := hashPassword(Str(q.Body()["new_password"]))
	q.Tx(func(tx *gorm.DB) error {
		var r models.ResetToken
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&r, "id = ? AND used = ? AND expires_at > ?", hex.EncodeToString(hash[:]), false, time.Now()).Error != nil || strconv.FormatInt(r.UserID, 10) != string(uid) {
			Fail(400, "Invalid or expired reset link")
		}
		if e := tx.Model(&models.User{}).Where("id = ?", r.UserID).Update("password", password).Error; e != nil {
			return e
		}
		return tx.Model(&r).Update("used", true).Error
	})
	return map[string]any{"message": "Password has been reset"}
}
