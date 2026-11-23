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

func (handler *Handler) OAuth(req *Request) any {
	body := req.Body()
	vk := strings.Contains(req.R.URL.Path, "/vk/")
	if vk {
		if handler.Config.VKClientID == "" || handler.Config.VKClientSecret == "" {
			Fail(503, "VK OAuth is not configured")
		}
	} else if handler.Config.GoogleClientID == "" {
		Fail(503, "Google OAuth is not configured")
	}
	var profile map[string]any
	if vk {
		code := Str(body["code"])
		if code == "" {
			Fail(400, "code is required")
		}
		data := handler.FetchJSON(req, "GET", "https://oauth.vk.com/access_token?"+url.Values{"client_id": {handler.Config.VKClientID}, "client_secret": {handler.Config.VKClientSecret}, "code": {code}, "redirect_uri": {handler.Config.FrontendURL + "/auth/vk/callback"}}.Encode(), nil, nil).(map[string]any)
		if data["error"] != nil || Str(data["access_token"]) == "" {
			Fail(401, "VK authentication failed")
		}
		result := handler.FetchJSON(req, "GET", "https://api.vk.com/method/get?"+url.Values{"access_token": {Str(data["access_token"])}, "user_ids": {Str(data["user_id"])}, "fields": {"photo_max"}, "v": {"5.199"}}.Encode(), nil, nil).(map[string]any)
		list, ok := result["response"].([]any)
		if !ok || len(list) == 0 {
			Fail(401, "VK profile is unavailable")
		}
		profileData := list[0].(map[string]any)
		email := Str(data["email"])
		if email == "" {
			email = "vk_" + Str(data["user_id"]) + "@oauth.litloop.invalid"
		}
		profile = map[string]any{"email": email, "picture": profileData["photo_max"], "email_verified": true}
	} else {
		idToken := Str(body["id_token"])
		if code := Str(body["code"]); code != "" {
			secret := os.Getenv("GOOGLE_CLIENT_SECRET")
			if secret == "" {
				Fail(503, "GOOGLE_CLIENT_SECRET is not configured")
			}
			redirect := strings.TrimRight(handler.Config.FrontendURL, "/") + "/auth/google/callback"
			if uri := Str(body["redirect_uri"]); uri != "" {
				if uri != redirect && !slices.Contains(handler.Config.GoogleAdditionalRedirectURIs, uri) {
					Fail(400, "Google redirect URI is not allowed")
				}
				redirect = uri
			}
			data := handler.FetchJSON(req, "POST", "https://oauth2.googleapis.com/token", []byte(url.Values{"client_id": {handler.Config.GoogleClientID}, "client_secret": {secret}, "code": {code}, "grant_type": {"authorization_code"}, "redirect_uri": {redirect}}.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}).(map[string]any)
			idToken = Str(data["id_token"])
		}
		if idToken == "" {
			Fail(400, "code or id_token is required")
		}
		profile = handler.FetchJSON(req, "GET", "https://oauth2.googleapis.com/tokeninfo?id_token="+url.QueryEscape(idToken), nil, nil).(map[string]any)
		if Str(profile["aud"]) != handler.Config.GoogleClientID || (Str(profile["email_verified"]) != "true") {
			Fail(401, "Google token is invalid")
		}
	}
	email := strings.ToLower(Str(profile["email"]))
	if email == "" {
		Fail(401, "Email is missing")
	}
	var user models.User
	created := false
	req.Tx(func(tx *gorm.DB) error {
		err := tx.First(&user, "email = ?", email).Error
		if err == gorm.ErrRecordNotFound {
			created = true
			user = models.User{Email: email, Username: strings.Split(email, "@")[0] + "_" + Token()[:6], Avatar: Str(profile["picture"]), Password: "!", IsVerified: true, IsActive: true}
			Check(tx.Create(&user).Error)
			chat := models.Chat{ChatType: "direct", Name: fmt.Sprintf("saved-%d", user.ID), IsSavedMessages: true}
			Check(tx.Create(&chat).Error)
			return tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: user.ID}).Error
		}
		return err
	})
	if !user.IsActive {
		Fail(403, "Inactive account")
	}
	out := handler.Tokens(user)
	out["created"] = created
	return out
}
