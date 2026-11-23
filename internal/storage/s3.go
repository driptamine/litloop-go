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
	Client	*s3.Client
	Bucket	string
}

func New(appConfig config.Config, provider string) (*Store, error) {
	endpoint, bucket, access, secret, region := appConfig.S3Endpoint, appConfig.S3Bucket, appConfig.S3AccessKey, appConfig.S3SecretKey, appConfig.S3Region
	switch provider {
	case "r2":
		endpoint, bucket, access, secret, region = appConfig.R2Endpoint, appConfig.R2Bucket, appConfig.R2AccessKey, appConfig.R2SecretKey, "auto"
		if endpoint == "" {
			return nil, fmt.Errorf("r2 endpoint is not configured")
		}
	case "gcs":
		endpoint, bucket, access, secret, region = "https://storage.googleapis.com", appConfig.GCSBucket, appConfig.GCSAccessKey, appConfig.GCSSecretKey, "auto"
	}
	if bucket == "" || access == "" || secret == "" {
		return nil, fmt.Errorf("%s storage is not configured", provider)
	}
	client := s3.New(s3.Options{Region: region, Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""), BaseEndpoint: optional(endpoint), UsePathStyle: provider != "s3" || endpoint != ""})
	return &Store{Client: client, Bucket: bucket}, nil
}
func optional(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}
func (store *Store) Initiate(ctx context.Context, key, typ string) (string, error) {
	output, err := store.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{Bucket: &store.Bucket, Key: &key, ContentType: &typ})
	if err != nil {
		return "", err
	}
	return aws.ToString(output.UploadId), nil
}
func (store *Store) PartURL(ctx context.Context, key, id string, partNumber int32) (string, error) {
	output, err := s3.NewPresignClient(store.Client).PresignUploadPart(ctx, &s3.UploadPartInput{Bucket: &store.Bucket, Key: &key, UploadId: &id, PartNumber: &partNumber}, s3.WithPresignExpires(time.Hour))
	if err != nil {
		return "", err
	}
	return output.URL, nil
}
func (store *Store) Complete(ctx context.Context, key, id string, parts []types.CompletedPart) error {
	_, err := store.Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{Bucket: &store.Bucket, Key: &key, UploadId: &id, MultipartUpload: &types.CompletedMultipartUpload{Parts: parts}})
	return err
}
func (store *Store) Abort(ctx context.Context, key, id string) error {
	_, err := store.Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: &store.Bucket, Key: &key, UploadId: &id})
	return err
}
func (store *Store) GetURL(ctx context.Context, key string) (string, error) {
	output, err := s3.NewPresignClient(store.Client).PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &store.Bucket, Key: &key}, s3.WithPresignExpires(time.Hour))
	if err != nil {
		return "", err
	}
	return output.URL, nil
}
func (store *Store) Exists(ctx context.Context, key string) error {
	_, err := store.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &store.Bucket, Key: &key})
	return err
}
func (store *Store) Put(ctx context.Context, key, typ string, body io.Reader) error {
	_, err := store.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: &store.Bucket, Key: &key, ContentType: &typ, Body: body})
	return err
}
