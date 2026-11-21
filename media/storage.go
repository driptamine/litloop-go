package media

import (
	"context"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/storage"
	"net/http"
	"strings"
)

func (s *Handler) NativeURL(key string) string      { return storage.NativeURL(s.Config, key) }
func (s *Handler) SignMedia(key, exp string) string { return storage.Signature(s.Config, key, exp) }
func (s *Handler) LocalURL(key string) string       { return storage.LocalURL(s.Config, key) }
func (s *Handler) MediaURL(r, g, a string) string   { return storage.MediaURL(s.Config, r, g, a) }
func (s *Handler) UploadProvider(q *Request) string {
	provider := s.legacyUploadProvider(q)
	if provider != "native" && s.Config.UsesR2ForMedia() {
		return "r2"
	}
	return provider
}

func (s *Handler) legacyUploadProvider(q *Request) string {
	if strings.Contains(q.R.URL.Path, "/r2/") {
		return "r2"
	}
	if strings.Contains(q.R.URL.Path, "/gcs/native/") {
		return "native"
	}
	if strings.Contains(q.R.URL.Path, "/gcs/") {
		return "gcs"
	}
	if s.Config.S3Bucket == "" {
		return "local"
	}
	return "s3"
}
func (s *Handler) StorageFor(provider string) *storage.Store {
	v, e := storage.New(s.Config, provider)
	if e != nil {
		Fail(503, e.Error())
	}
	return v
}
func (s *Handler) GoogleClient(ctx context.Context) *http.Client {
	client, e := storage.NativeClient(ctx, s.Config)
	if e != nil {
		Fail(503, "Native GCS is not configured")
	}
	return client
}
