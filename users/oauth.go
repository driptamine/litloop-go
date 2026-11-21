package users

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/url"
	"os"
	"slices"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) OAuth(q *Request) any {
	b := q.Body()
	vk := strings.Contains(q.R.URL.Path, "/vk/")
	if vk {
		if s.Config.VKClientID == "" || s.Config.VKClientSecret == "" {
			Fail(503, "VK OAuth is not configured")
		}
	} else if s.Config.GoogleClientID == "" {
		Fail(503, "Google OAuth is not configured")
	}
	var profile map[string]any
	if vk {
		code := Str(b["code"])
		if code == "" {
			Fail(400, "code is required")
		}
		data := s.FetchJSON(q, "GET", "https://oauth.vk.com/access_token?"+url.Values{"client_id": {s.Config.VKClientID}, "client_secret": {s.Config.VKClientSecret}, "code": {code}, "redirect_uri": {s.Config.FrontendURL + "/auth/vk/callback"}}.Encode(), nil, nil).(map[string]any)
		if data["error"] != nil || Str(data["access_token"]) == "" {
			Fail(401, "VK authentication failed")
		}
		result := s.FetchJSON(q, "GET", "https://api.vk.com/method/get?"+url.Values{"access_token": {Str(data["access_token"])}, "user_ids": {Str(data["user_id"])}, "fields": {"photo_max"}, "v": {"5.199"}}.Encode(), nil, nil).(map[string]any)
		list, ok := result["response"].([]any)
		if !ok || len(list) == 0 {
			Fail(401, "VK profile is unavailable")
		}
		p := list[0].(map[string]any)
		email := Str(data["email"])
		if email == "" {
			email = "vk_" + Str(data["user_id"]) + "@oauth.litloop.invalid"
		}
		profile = map[string]any{"email": email, "picture": p["photo_max"], "email_verified": true}
	} else {
		idToken := Str(b["id_token"])
		if code := Str(b["code"]); code != "" {
			secret := os.Getenv("GOOGLE_CLIENT_SECRET")
			if secret == "" {
				Fail(503, "GOOGLE_CLIENT_SECRET is not configured")
			}
			redirect := strings.TrimRight(s.Config.FrontendURL, "/") + "/auth/google/callback"
			if uri := Str(b["redirect_uri"]); uri != "" {
				if uri != redirect && !slices.Contains(s.Config.GoogleAdditionalRedirectURIs, uri) {
					Fail(400, "Google redirect URI is not allowed")
				}
				redirect = uri
			}
			data := s.FetchJSON(q, "POST", "https://oauth2.googleapis.com/token", []byte(url.Values{"client_id": {s.Config.GoogleClientID}, "client_secret": {secret}, "code": {code}, "grant_type": {"authorization_code"}, "redirect_uri": {redirect}}.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}).(map[string]any)
			idToken = Str(data["id_token"])
		}
		if idToken == "" {
			Fail(400, "code or id_token is required")
		}
		profile = s.FetchJSON(q, "GET", "https://oauth2.googleapis.com/tokeninfo?id_token="+url.QueryEscape(idToken), nil, nil).(map[string]any)
		if Str(profile["aud"]) != s.Config.GoogleClientID || (Str(profile["email_verified"]) != "true") {
			Fail(401, "Google token is invalid")
		}
	}
	email := strings.ToLower(Str(profile["email"]))
	if email == "" {
		Fail(401, "Email is missing")
	}
	var u models.User
	created := false
	q.Tx(func(tx *gorm.DB) error {
		e := tx.First(&u, "email = ?", email).Error
		if e == gorm.ErrRecordNotFound {
			created = true
			u = models.User{Email: email, Username: strings.Split(email, "@")[0] + "_" + Token()[:6], Avatar: Str(profile["picture"]), Password: "!", IsVerified: true, IsActive: true}
			Check(tx.Create(&u).Error)
			chat := models.Chat{ChatType: "direct", Name: fmt.Sprintf("saved-%d", u.ID), IsSavedMessages: true}
			Check(tx.Create(&chat).Error)
			return tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: u.ID}).Error
		}
		return e
	})
	if !u.IsActive {
		Fail(403, "Inactive account")
	}
	out := s.Tokens(u)
	out["created"] = created
	return out
}
