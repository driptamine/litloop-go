package media

import (
	"context"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/storage"
	"net/http"
	"strings"
)

func (handler *Handler) NativeURL(key string) string	{ return storage.NativeURL(handler.Config, key) }
func (handler *Handler) SignMedia(key, exp string) string {
	return storage.Signature(handler.Config, key, exp)
}
func (handler *Handler) LocalURL(key string) string	{ return storage.LocalURL(handler.Config, key) }
func (handler *Handler) MediaURL(r2Key, gcsKey, s3Key string) string {
	return storage.MediaURL(handler.Config, r2Key, gcsKey, s3Key)
}
func (handler *Handler) UploadProvider(req *Request) string {
	provider := handler.legacyUploadProvider(req)
	if provider != "native" && handler.Config.UsesR2ForMedia() {
		return "r2"
	}
	return provider
}

func (handler *Handler) legacyUploadProvider(req *Request) string {
	if strings.Contains(req.R.URL.Path, "/r2/") {
		return "r2"
	}
	if strings.Contains(req.R.URL.Path, "/gcs/native/") {
		return "native"
	}
	if strings.Contains(req.R.URL.Path, "/gcs/") {
		return "gcs"
	}
	if handler.Config.S3Bucket == "" {
		return "local"
	}
	return "s3"
}
func (handler *Handler) StorageFor(provider string) *storage.Store {
	store, err := storage.New(handler.Config, provider)
	if err != nil {
		Fail(503, err.Error())
	}
	return store
}
func (handler *Handler) GoogleClient(ctx context.Context) *http.Client {
	client, err := storage.NativeClient(ctx, handler.Config)
	if err != nil {
		Fail(503, "Native GCS is not configured")
	}
	return client
}
