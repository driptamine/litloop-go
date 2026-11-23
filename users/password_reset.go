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

func (handler *Handler) ForgotPassword(req *Request) any {
	email := strings.ToLower(Str(req.Body()["email"]))
	if email == "" {
		Fail(400, "Email is required")
	}
	if handler.Config.SMTPAddr == "" {
		Fail(503, "SMTP is not configured")
	}
	var user models.User
	if handler.DB.First(&user, "email = ?", email).Error == nil {
		raw := Token()
		sum := sha256.Sum256([]byte(raw))
		Check(handler.DB.Create(&models.ResetToken{ID: hex.EncodeToString(sum[:]), UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}).Error)
		uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(user.ID, 10)))
		link := strings.TrimRight(handler.Config.FrontendURL, "/") + "/password-reset/" + uid + "/" + raw
		Check(sendMail(handler.Config.SMTPAddr, handler.Config.SMTPUser, handler.Config.SMTPPassword, handler.Config.SMTPFrom, user.Email, "Password reset", link))
	}
	return map[string]any{"message": "If an account exists, a reset link has been sent."}
}
func sendMail(addr, user, password, from, to, subject, body string) error {
	host := strings.Split(addr, ":")[0]
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}
	for _, address := range []string{from, to, subject} {
		if strings.ContainsAny(address, "\r\n") {
			return fmt.Errorf("invalid mail header")
		}
	}
	msg := []byte("From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
func (handler *Handler) ResetPassword(req *Request) any {
	uid, err := base64.RawURLEncoding.DecodeString(req.Param("uid"))
	if err != nil {
		Fail(400, "Invalid reset link")
	}
	hash := sha256.Sum256([]byte(req.Param("token")))
	password := hashPassword(Str(req.Body()["new_password"]))
	req.Tx(func(tx *gorm.DB) error {
		var resetToken models.ResetToken
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&resetToken, "id = ? AND used = ? AND expires_at > ?", hex.EncodeToString(hash[:]), false, time.Now()).Error != nil || strconv.FormatInt(resetToken.UserID, 10) != string(uid) {
			Fail(400, "Invalid or expired reset link")
		}
		if err := tx.Model(&models.User{}).Where("id = ?", resetToken.UserID).Update("password", password).Error; err != nil {
			return err
		}
		return tx.Model(&resetToken).Update("used", true).Error
	})
	return map[string]any{"message": "Password has been reset"}
}
