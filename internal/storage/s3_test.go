package storage

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"litloop/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMultipartS3Contract(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		w.Header().Set("Content-Type", "application/xml")
		if r.Header.Get("Authorization") == "" {
			t.Error("unsigned S3 request")
		}
		switch {
		case r.Method == "POST" && r.URL.Query().Has("uploads"):
			_, _ = w.Write([]byte(`<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>videos/test.mp4</Key><UploadId>remote-123</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == "POST" && r.URL.Query().Get("uploadId") == "remote-123":
			_, _ = w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>videos/test.mp4</Key><ETag>done</ETag></CompleteMultipartUploadResult>`))
		case r.Method == "HEAD":
			w.WriteHeader(200)
		default:
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	store, e := New(config.Config{R2Endpoint: server.URL, R2Bucket: "bucket", R2AccessKey: "test", R2SecretKey: "secret"}, "r2")
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	id, e := store.Initiate(ctx, "videos/test.mp4", "video/mp4")
	if e != nil || id != "remote-123" {
		t.Fatal(e, id)
	}
	signed, e := store.PartURL(ctx, "videos/test.mp4", id, 1)
	if e != nil {
		t.Fatal(e)
	}
	parsed, _ := url.Parse(signed)
	if parsed.Query().Get("uploadId") != id || parsed.Query().Get("partNumber") != "1" || !strings.Contains(signed, "X-Amz-Signature") {
		t.Fatal(signed)
	}
	if e = store.Complete(ctx, "videos/test.mp4", id, []types.CompletedPart{{PartNumber: aws.Int32(1), ETag: aws.String("etag")}}); e != nil {
		t.Fatal(e)
	}
	if e = store.Exists(ctx, "videos/test.mp4"); e != nil {
		t.Fatal(e)
	}
	if len(calls) != 3 {
		t.Fatal(calls)
	}
}
