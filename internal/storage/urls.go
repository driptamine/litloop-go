package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"litloop/backend/internal/config"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Signature(appConfig config.Config, key, expires string) string {
	mac := hmac.New(sha256.New, []byte(appConfig.JWTSecret))
	_, _ = mac.Write([]byte(key + "\n" + expires))
	return hex.EncodeToString(mac.Sum(nil))
}
func LocalURL(appConfig config.Config, key string) string {
	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	path := (&url.URL{Path: "/media/" + key}).EscapedPath()
	return strings.TrimRight(appConfig.PublicURL, "/") + path + "?expires=" + exp + "&signature=" + Signature(appConfig, key, exp)
}

func ChatAttachmentPrefix(userID int64) string {
	return "chat-attachments/" + strconv.FormatInt(userID, 10) + "/"
}
func NativeURL(appConfig config.Config, key string) string {
	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	path := (&url.URL{Path: "/gcs-media/" + key}).EscapedPath()
	return strings.TrimRight(appConfig.PublicURL, "/") + path + "?expires=" + exp + "&signature=" + Signature(appConfig, "gcs/"+key, exp)
}
func MediaURL(appConfig config.Config, r2, gcs, s3key string) string {
	key, provider := r2, "r2"
	if key == "" {
		key, provider = gcs, "gcs"
	}
	if key == "" {
		key, provider = s3key, "s3"
	}
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return key
	}
	if strings.HasPrefix(key, "local/") {
		return LocalURL(appConfig, strings.TrimPrefix(key, "local/"))
	}
	if provider == "gcs" && appConfig.GCSAccessKey == "" && appConfig.GCSBucket != "" {
		return NativeURL(appConfig, key)
	}
	public := ""
	if provider == "r2" {
		public = appConfig.R2PublicURL
	} else if provider == "s3" {
		public = appConfig.S3PublicURL
	}
	if public != "" {
		return strings.TrimRight(public, "/") + "/" + key
	}
	return PresignedURL(appConfig, provider, key)
}

func PresignedURL(appConfig config.Config, provider, key string) string {
	store, err := New(appConfig, provider)
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	uri, err := store.GetURL(ctx, key)
	if err != nil {
		return ""
	}
	return uri
}
