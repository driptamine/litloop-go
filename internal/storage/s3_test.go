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

func TestMultipartS3Contract(tester *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpReq *http.Request) {
		calls = append(calls, httpReq.Method)
		writer.Header().Set("Content-Type", "application/xml")
		if httpReq.Header.Get("Authorization") == "" {
			tester.Error("unsigned S3 request")
		}
		switch {
		case httpReq.Method == "POST" && httpReq.URL.Query().Has("uploads"):
			_, _ = writer.Write([]byte(`<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>videos/test.mp4</Key><UploadId>remote-123</UploadId></InitiateMultipartUploadResult>`))
		case httpReq.Method == "POST" && httpReq.URL.Query().Get("uploadId") == "remote-123":
			_, _ = writer.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>videos/test.mp4</Key><ETag>done</ETag></CompleteMultipartUploadResult>`))
		case httpReq.Method == "HEAD":
			writer.WriteHeader(200)
		default:
			writer.WriteHeader(400)
		}
	}))
	defer server.Close()
	store, err := New(config.Config{R2Endpoint: server.URL, R2Bucket: "bucket", R2AccessKey: "test", R2SecretKey: "secret"}, "r2")
	if err != nil {
		tester.Fatal(err)
	}
	ctx := context.Background()
	id, err := store.Initiate(ctx, "videos/test.mp4", "video/mp4")
	if err != nil || id != "remote-123" {
		tester.Fatal(err, id)
	}
	signed, err := store.PartURL(ctx, "videos/test.mp4", id, 1)
	if err != nil {
		tester.Fatal(err)
	}
	parsed, _ := url.Parse(signed)
	if parsed.Query().Get("uploadId") != id || parsed.Query().Get("partNumber") != "1" || !strings.Contains(signed, "X-Amz-Signature") {
		tester.Fatal(signed)
	}
	if err = store.Complete(ctx, "videos/test.mp4", id, []types.CompletedPart{{PartNumber: aws.Int32(1), ETag: aws.String("etag")}}); err != nil {
		tester.Fatal(err)
	}
	if err = store.Exists(ctx, "videos/test.mp4"); err != nil {
		tester.Fatal(err)
	}
	if len(calls) != 3 {
		tester.Fatal(calls)
	}
}
