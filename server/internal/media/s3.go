package media

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	DefaultBucket    = "fproto-media"
	UploadExpiry     = 1 * time.Hour
	DownloadExpiry   = 24 * time.Hour
)

// S3Client wraps minio-go for presigned URL generation.
type S3Client struct {
	client *minio.Client
	bucket string
}

// NewS3Client creates a MinIO/S3 client.
func NewS3Client(endpoint, accessKey, secretKey string, useSSL bool) (*S3Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &S3Client{client: client, bucket: DefaultBucket}, nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func (s *S3Client) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("bucket exists check: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("make bucket: %w", err)
		}
	}
	return nil
}

// PresignedPutURL generates a presigned PUT URL for uploading an object.
func (s *S3Client) PresignedPutURL(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, UploadExpiry)
	if err != nil {
		return "", fmt.Errorf("presigned put: %w", err)
	}
	return u.String(), nil
}

// PresignedGetURL generates a presigned GET URL for downloading an object.
func (s *S3Client) PresignedGetURL(ctx context.Context, key string) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, DownloadExpiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("presigned get: %w", err)
	}
	return u.String(), nil
}

// ObjectExists checks if an object exists in the bucket.
func (s *S3Client) ObjectExists(ctx context.Context, key string) (bool, int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, info.Size, nil
}
