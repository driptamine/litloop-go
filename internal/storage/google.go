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

func NativeReadClient(ctx context.Context, appConfig config.Config) (*http.Client, error) {
	if appConfig.GoogleCredentials != "" {
		return NativeClient(ctx, appConfig)
	}
	if appConfig.GCSBucket == "" {
		return nil, fmt.Errorf("native GCS is not configured")
	}
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/devstorage.read_only")
	if err != nil {
		return nil, err
	}
	client.Timeout = 5 * time.Minute
	return client, nil
}

func NativeClient(ctx context.Context, appConfig config.Config) (*http.Client, error) {
	if appConfig.GCSBucket == "" || appConfig.GoogleCredentials == "" {
		return nil, fmt.Errorf("native GCS is not configured")
	}
	data, err := os.ReadFile(appConfig.GoogleCredentials)
	if err != nil {
		return nil, err
	}
	jwt, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return nil, err
	}
	client := jwt.Client(ctx)
	client.Timeout = 5 * time.Minute
	return client, nil
}
