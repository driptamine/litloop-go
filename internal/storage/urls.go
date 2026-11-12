package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"litloop/backend/internal/config"
	"strconv"
	"strings"
	"time"
)

func Signature(c config.Config, key, expires string) string {
	m := hmac.New(sha256.New, []byte(c.JWTSecret))
	_, _ = m.Write([]byte(key + "\n" + expires))
	return hex.EncodeToString(m.Sum(nil))
}
func LocalURL(c config.Config, key string) string {
	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	return strings.TrimRight(c.PublicURL, "/") + "/media/" + key + "?expires=" + exp + "&signature=" + Signature(c, key, exp)
}
func NativeURL(c config.Config, key string) string {
	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	return strings.TrimRight(c.PublicURL, "/") + "/gcs-media/" + key + "?expires=" + exp + "&signature=" + Signature(c, "gcs/"+key, exp)
}
func MediaURL(c config.Config, r2, gcs, s3key string) string {
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
		return LocalURL(c, strings.TrimPrefix(key, "local/"))
	}
	if provider == "gcs" && c.GCSAccessKey == "" && c.GoogleCredentials != "" {
		return NativeURL(c, key)
	}
	public := ""
	if provider == "r2" {
		public = c.R2PublicURL
	} else if provider == "s3" {
		public = c.S3PublicURL
	}
	if public != "" {
		return strings.TrimRight(public, "/") + "/" + key
	}
	store, e := New(c, provider)
	if e != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	uri, e := store.GetURL(ctx, key)
	if e != nil {
		return ""
	}
	return uri
}
