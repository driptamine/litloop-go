package storage

import (
	"net/url"
	"testing"

	"litloop/backend/internal/config"
)

func TestGCSMediaURLWithApplicationDefaultCredentials(t *testing.T) {
	cfg := config.Config{
		GCSBucket: "legacy-media",
		PublicURL: "https://api.example.com",
		JWTSecret: "test-secret",
	}
	key := "album/Track #1? 100%.mp3"
	uri, err := url.Parse(MediaURL(cfg, "", key, ""))
	if err != nil {
		t.Fatal(err)
	}
	if uri.Host != "api.example.com" || uri.Path != "/gcs-media/"+key || uri.Fragment != "" {
		t.Fatalf("URL does not address the GCS proxy for %q", key)
	}
	expires := uri.Query().Get("expires")
	if expires == "" || uri.Query().Get("signature") != Signature(cfg, "gcs/"+key, expires) {
		t.Fatal("URL is missing a valid media signature")
	}
}
