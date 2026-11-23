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

func (roundTrip roundTripFunc) RoundTrip(httpReq *http.Request) (*http.Response, error) {
	return roundTrip(httpReq)
}
func response(value any) *http.Response {
	encoded, _ := json.Marshal(value)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(encoded))}
}
func TestExternalAdapters(tester *testing.T) {
	fixture := setup(tester)
	fixture.s.Config.GoogleClientID = "client-id"
	fixture.s.Config.GeminiKey = "fake-key"
	fixture.s.Client.Transport = roundTripFunc(func(httpReq *http.Request) (*http.Response, error) {
		switch httpReq.URL.Host {
		case "oauth2.googleapis.com":
			return response(map[string]any{"aud": "client-id", "email_verified": "true", "email": "oauth@example.com", "picture": "https://example.com/avatar.png"}), nil
		case "suggestqueries.google.com":
			return response([]any{"go", []string{"golang", "gorm"}}), nil
		case "generativelanguage.googleapis.com":
			if httpReq.Header.Get("x-goog-api-key") != "fake-key" {
				tester.Error("Missing Gemini key")
			}
			return response(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": "Hello"}}}}}}), nil
		}
		return nil, fmt.Errorf("unexpected external request %s", httpReq.URL)
	})
	OAuth := fixture.req("POST", "/auth/google/token", "", map[string]any{"id_token": "verified-by-google"}, 200)
	if OAuth["created"] != true {
		tester.Fatal(OAuth)
	}
	fixture.req("GET", "/users/me", OAuth["access"].(string), nil, 200)
	suggestions := fixture.req("GET", "/queries/google/search?q=go", "", nil, 200)
	if len(suggestions["results"].([]any)) != 2 {
		tester.Fatal(suggestions)
	}
	var count int64
	if err := fixture.s.DB.Model(&models.GoogleQuery{}).Count(&count).Error; err != nil || count != 2 {
		tester.Fatal(err, count)
	}
	ai := fixture.req("GET", "/chats/gemini?q=Hi", fixture.Token, nil, 200)
	if ai["response"] != "Hello" {
		tester.Fatal(ai)
	}
	if err := fixture.s.DB.Model(&models.Message{}).Where("chat_id = ?", ai["chat_id"]).Count(&count).Error; err != nil || count != 2 {
		tester.Fatal(err, count)
	}
	fixture.s.Client.Transport = roundTripFunc(func(httpReq *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("rate limit")), Header: http.Header{}}, nil
	})
	fixture.req("GET", "/chats/gemini?q=Hi", fixture.Token, nil, 502)
}

func TestGoogleCodeExchangeRedirectURIs(tester *testing.T) {
	fixture := setup(tester)
	tester.Setenv("GOOGLE_CLIENT_SECRET", "test-google-secret")
	const primary = "https://litloop.netlify.app/auth/google/callback"
	const layero = "https://litloop.layero.app/auth/google/callback"
	fixture.s.Config.FrontendURL = "https://litloop.netlify.app"
	fixture.s.Config.GoogleClientID = "client-id"
	fixture.s.Config.GoogleAdditionalRedirectURIs = []string{layero}
	var redirects []string
	fixture.s.Client.Transport = roundTripFunc(func(httpReq *http.Request) (*http.Response, error) {
		if httpReq.URL.Host == "oauth2.googleapis.com" && httpReq.URL.Path == "/token" {
			if err := httpReq.ParseForm(); err != nil {
				return nil, err
			}
			redirects = append(redirects, httpReq.Form.Get("redirect_uri"))
			return response(map[string]any{"id_token": "test-id-token"}), nil
		}
		if httpReq.URL.Host == "oauth2.googleapis.com" && httpReq.URL.Path == "/tokeninfo" {
			return response(map[string]any{"aud": "client-id", "email_verified": true, "email": "oauth-layero@example.test"}), nil
		}
		return nil, fmt.Errorf("unexpected Google endpoint %s", httpReq.URL.Path)
	})
	for _, uri := range []string{primary, layero} {
		fixture.req("POST", "/auth/google/token", "", map[string]any{"code": "test-code", "redirect_uri": uri}, 200)
	}
	fixture.req("POST", "/auth/google/token", "", map[string]any{"code": "test-code", "redirect_uri": layero + "?unexpected=1"}, 400)
	if len(redirects) != 2 || redirects[0] != primary || redirects[1] != layero {
		tester.Fatal(redirects)
	}
}

func TestCSVImportsAndOwnership(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	upload := func(path, filename, content string) {
		tester.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("csv_file", filename)
		if err != nil {
			tester.Fatal(err)
		}
		_, _ = io.WriteString(part, content)
		_ = writer.Close()
		req, _ := http.NewRequest("POST", fixture.http.URL+path, &body)
		req.Header.Set("Authorization", "Bearer "+fixture.Token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		res, err := fixture.http.Client().Do(req)
		if err != nil {
			tester.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			tester.Fatalf("%s: %d %s", path, res.StatusCode, raw)
		}
	}
	upload("/links/upload-csv", "links.csv", "url,title,tags,time_added\nhttps://example.com,Example,go|books,1700000000\n")
	Links := fixture.req("GET", "/links/all", fixture.Token, nil, 200)
	if len(Links["results"].([]any)) != 1 {
		tester.Fatal(Links)
	}
	other := fixture.req("GET", "/links/all", bob, nil, 200)
	if len(other["results"].([]any)) != 0 {
		tester.Fatal(other)
	}
	upload("/movies/import_watchlist_csv", "watchlist.csv", "Const,Title,IMDb Rating,Year\ntt0133093,The Matrix,8.7,1999\n")
	upload("/movies/import_ratings_csv", "ratings.csv", "Const,Title,Your Rating,Date Rated\ntt0133093,The Matrix,9,2026-11-12\n")
	watch := fixture.req("GET", "/movies/watchlist", fixture.Token, nil, 200)
	if len(watch["watchlist"].([]any)) != 1 {
		tester.Fatal(watch)
	}
	ratings := fixture.req("GET", "/movies/alice/ratings", "", nil, 200)
	if len(ratings["ratings"].([]any)) != 1 {
		tester.Fatal(ratings)
	}
	fixture.req("GET", "/users/alice/watchlist", bob, nil, 403)
}
func TestResetTokenSingleUse(tester *testing.T) {
	fixture := setup(tester)
	raw := "one-use-reset-token"
	sum := sha256.Sum256([]byte(raw))
	encodedUser := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprint(fixture.user)))
	if err := fixture.s.DB.Create(&models.ResetToken{ID: hex.EncodeToString(sum[:]), UserID: fixture.user, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		tester.Fatal(err)
	}
	path := "/users/password-reset-confirm/" + encodedUser + "/" + raw
	fixture.req("POST", path, "", map[string]any{"new_password": "new-password"}, 200)
	fixture.req("POST", path, "", map[string]any{"new_password": "another-password"}, 400)
	fixture.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "new-password"}, 200)
}
