package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"litloop/backend/internal/models"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(b))}
}
func TestExternalAdapters(t *testing.T) {
	f := setup(t)
	f.s.Config.GoogleClientID = "client-id"
	f.s.Config.GeminiKey = "fake-key"
	f.s.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "oauth2.googleapis.com":
			return response(map[string]any{"aud": "client-id", "email_verified": "true", "email": "oauth@example.com", "picture": "https://example.com/avatar.png"}), nil
		case "suggestqueries.google.com":
			return response([]any{"go", []string{"golang", "gorm"}}), nil
		case "generativelanguage.googleapis.com":
			if r.Header.Get("x-goog-api-key") != "fake-key" {
				t.Error("Missing Gemini key")
			}
			return response(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": "Hello"}}}}}}), nil
		}
		return nil, fmt.Errorf("unexpected external request %s", r.URL)
	})
	OAuth := f.req("POST", "/auth/google/token", "", map[string]any{"id_token": "verified-by-google"}, 200)
	if OAuth["created"] != true {
		t.Fatal(OAuth)
	}
	f.req("GET", "/users/me", OAuth["access"].(string), nil, 200)
	suggestions := f.req("GET", "/queries/google/search?q=go", "", nil, 200)
	if len(suggestions["results"].([]any)) != 2 {
		t.Fatal(suggestions)
	}
	var count int64
	if e := f.s.DB.Model(&models.GoogleQuery{}).Count(&count).Error; e != nil || count != 2 {
		t.Fatal(e, count)
	}
	ai := f.req("GET", "/chats/gemini?q=Hi", f.Token, nil, 200)
	if ai["response"] != "Hello" {
		t.Fatal(ai)
	}
	if e := f.s.DB.Model(&models.Message{}).Where("chat_id = ?", ai["chat_id"]).Count(&count).Error; e != nil || count != 2 {
		t.Fatal(e, count)
	}
	f.s.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("rate limit")), Header: http.Header{}}, nil
	})
	f.req("GET", "/chats/gemini?q=Hi", f.Token, nil, 502)
}

func TestGoogleCodeExchangeRedirectURIs(t *testing.T) {
	f := setup(t)
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-google-secret")
	const primary = "https://litloop.netlify.app/auth/google/callback"
	const layero = "https://litloop.layero.app/auth/google/callback"
	f.s.Config.FrontendURL = "https://litloop.netlify.app"
	f.s.Config.GoogleClientID = "client-id"
	f.s.Config.GoogleAdditionalRedirectURIs = []string{layero}
	var redirects []string
	f.s.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "oauth2.googleapis.com" && r.URL.Path == "/token" {
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			redirects = append(redirects, r.Form.Get("redirect_uri"))
			return response(map[string]any{"id_token": "test-id-token"}), nil
		}
		if r.URL.Host == "oauth2.googleapis.com" && r.URL.Path == "/tokeninfo" {
			return response(map[string]any{"aud": "client-id", "email_verified": true, "email": "oauth-layero@example.test"}), nil
		}
		return nil, fmt.Errorf("unexpected Google endpoint %s", r.URL.Path)
	})
	for _, uri := range []string{primary, layero} {
		f.req("POST", "/auth/google/token", "", map[string]any{"code": "test-code", "redirect_uri": uri}, 200)
	}
	f.req("POST", "/auth/google/token", "", map[string]any{"code": "test-code", "redirect_uri": layero + "?unexpected=1"}, 400)
	if len(redirects) != 2 || redirects[0] != primary || redirects[1] != layero {
		t.Fatal(redirects)
	}
}

func TestCSVImportsAndOwnership(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	upload := func(path, filename, content string) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, e := writer.CreateFormFile("csv_file", filename)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = io.WriteString(part, content)
		_ = writer.Close()
		req, _ := http.NewRequest("POST", f.http.URL+path, &body)
		req.Header.Set("Authorization", "Bearer "+f.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		res, e := f.http.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("%s: %d %s", path, res.StatusCode, raw)
		}
	}
	upload("/links/upload-csv", "links.csv", "url,title,tags,time_added\nhttps://example.com,Example,go|books,1700000000\n")
	Links := f.req("GET", "/links/all", f.Token, nil, 200)
	if len(Links["results"].([]any)) != 1 {
		t.Fatal(Links)
	}
	other := f.req("GET", "/links/all", bob, nil, 200)
	if len(other["results"].([]any)) != 0 {
		t.Fatal(other)
	}
	upload("/movies/import_watchlist_csv", "watchlist.csv", "Const,Title,IMDb Rating,Year\ntt0133093,The Matrix,8.7,1999\n")
	upload("/movies/import_ratings_csv", "ratings.csv", "Const,Title,Your Rating,Date Rated\ntt0133093,The Matrix,9,2026-11-12\n")
	watch := f.req("GET", "/movies/watchlist", f.Token, nil, 200)
	if len(watch["watchlist"].([]any)) != 1 {
		t.Fatal(watch)
	}
	ratings := f.req("GET", "/movies/alice/ratings", "", nil, 200)
	if len(ratings["ratings"].([]any)) != 1 {
		t.Fatal(ratings)
	}
	f.req("GET", "/users/alice/watchlist", bob, nil, 403)
}
func TestResetTokenSingleUse(t *testing.T) {
	f := setup(t)
	raw := "one-use-reset-token"
	sum := sha256.Sum256([]byte(raw))
	u := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprint(f.user)))
	if e := f.s.DB.Create(&models.ResetToken{ID: hex.EncodeToString(sum[:]), UserID: f.user, ExpiresAt: time.Now().Add(time.Hour)}).Error; e != nil {
		t.Fatal(e)
	}
	path := "/users/password-reset-confirm/" + u + "/" + raw
	f.req("POST", path, "", map[string]any{"new_password": "new-password"}, 200)
	f.req("POST", path, "", map[string]any{"new_password": "another-password"}, 400)
	f.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "new-password"}, 200)
}
