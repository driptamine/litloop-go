package storage

import (
	"net/url"
	"testing"

	"litloop/backend/internal/config"
)

func TestGCSMediaURLWithApplicationDefaultCredentials(tester *testing.T) {
	cfg := config.Config{
		GCSBucket:	"legacy-media",
		PublicURL:	"https://api.example.com",
		JWTSecret:	"test-secret",
	}
	key := "album/Track #1? 100%.mp3"
	uri, err := url.Parse(MediaURL(cfg, "", key, ""))
	if err != nil {
		tester.Fatal(err)
	}
	if uri.Host != "api.example.com" || uri.Path != "/gcs-media/"+key || uri.Fragment != "" {
		tester.Fatalf("URL does not address the GCS proxy for %q", key)
	}
	expires := uri.Query().Get("expires")
	if expires == "" || uri.Query().Get("signature") != Signature(cfg, "gcs/"+key, expires) {
		tester.Fatal("URL is missing a valid media signature")
	}
}
