package storage

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"litloop/backend/internal/config"
	"time"
)

type Store struct {
	Client *s3.Client
	Bucket string
}

func New(c config.Config, provider string) (*Store, error) {
	endpoint, bucket, access, secret, region := c.S3Endpoint, c.S3Bucket, c.S3AccessKey, c.S3SecretKey, c.S3Region
	switch provider {
	case "r2":
		endpoint, bucket, access, secret, region = c.R2Endpoint, c.R2Bucket, c.R2AccessKey, c.R2SecretKey, "auto"
		if endpoint == "" {
			return nil, fmt.Errorf("r2 endpoint is not configured")
		}
	case "gcs":
		endpoint, bucket, access, secret, region = "https://storage.googleapis.com", c.GCSBucket, c.GCSAccessKey, c.GCSSecretKey, "auto"
	}
	if bucket == "" || access == "" || secret == "" {
		return nil, fmt.Errorf("%s storage is not configured", provider)
	}
	client := s3.New(s3.Options{Region: region, Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""), BaseEndpoint: optional(endpoint), UsePathStyle: provider != "s3" || endpoint != ""})
	return &Store{Client: client, Bucket: bucket}, nil
}
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func (s *Store) Initiate(ctx context.Context, key, typ string) (string, error) {
	o, e := s.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{Bucket: &s.Bucket, Key: &key, ContentType: &typ})
	if e != nil {
		return "", e
	}
	return aws.ToString(o.UploadId), nil
}
func (s *Store) PartURL(ctx context.Context, key, id string, n int32) (string, error) {
	o, e := s3.NewPresignClient(s.Client).PresignUploadPart(ctx, &s3.UploadPartInput{Bucket: &s.Bucket, Key: &key, UploadId: &id, PartNumber: &n}, s3.WithPresignExpires(time.Hour))
	if e != nil {
		return "", e
	}
	return o.URL, nil
}
func (s *Store) Complete(ctx context.Context, key, id string, parts []types.CompletedPart) error {
	_, e := s.Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{Bucket: &s.Bucket, Key: &key, UploadId: &id, MultipartUpload: &types.CompletedMultipartUpload{Parts: parts}})
	return e
}
func (s *Store) Abort(ctx context.Context, key, id string) error {
	_, e := s.Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: &s.Bucket, Key: &key, UploadId: &id})
	return e
}
func (s *Store) GetURL(ctx context.Context, key string) (string, error) {
	o, e := s3.NewPresignClient(s.Client).PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &key}, s3.WithPresignExpires(time.Hour))
	if e != nil {
		return "", e
	}
	return o.URL, nil
}
func (s *Store) Exists(ctx context.Context, key string) error {
	_, e := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.Bucket, Key: &key})
	return e
}
func (s *Store) Put(ctx context.Context, key, typ string, body io.Reader) error {
	_, e := s.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.Bucket, Key: &key, ContentType: &typ, Body: body})
	return e
}
