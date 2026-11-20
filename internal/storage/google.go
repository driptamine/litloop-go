package storage

import (
	"context"
	"fmt"
	"golang.org/x/oauth2/google"
	"litloop/backend/internal/config"
	"net/http"
	"os"
	"time"
)

func NativeReadClient(ctx context.Context, c config.Config) (*http.Client, error) {
	if c.GoogleCredentials != "" {
		return NativeClient(ctx, c)
	}
	if c.GCSBucket == "" {
		return nil, fmt.Errorf("native GCS is not configured")
	}
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/devstorage.read_only")
	if err != nil {
		return nil, err
	}
	client.Timeout = 5 * time.Minute
	return client, nil
}

func NativeClient(ctx context.Context, c config.Config) (*http.Client, error) {
	if c.GCSBucket == "" || c.GoogleCredentials == "" {
		return nil, fmt.Errorf("native GCS is not configured")
	}
	data, e := os.ReadFile(c.GoogleCredentials)
	if e != nil {
		return nil, e
	}
	jwt, e := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/devstorage.read_write")
	if e != nil {
		return nil, e
	}
	client := jwt.Client(ctx)
	client.Timeout = 5 * time.Minute
	return client, nil
}
