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
