package storage

import (
	"context"
	"fmt"
	"io"
)

// S3Storage implements StorageBackend using AWS S3.
// This is a placeholder that compiles but requires AWS SDK configuration to function.
type S3Storage struct {
	Bucket string
	Region string
}

// NewS3Storage creates a new S3Storage instance.
func NewS3Storage(bucket, region string) (*S3Storage, error) {
	if bucket == "" || region == "" {
		return nil, fmt.Errorf("S3 bucket and region are required")
	}
	return &S3Storage{Bucket: bucket, Region: region}, nil
}

func (s *S3Storage) Save(_ context.Context, _ string, _ io.Reader) error {
	// TODO: Implement with AWS SDK v2 when S3 credentials are configured.
	return fmt.Errorf("S3 storage not yet implemented")
}

func (s *S3Storage) URL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.Bucket, s.Region, key)
}

func (s *S3Storage) Delete(_ context.Context, _ string) error {
	// TODO: Implement with AWS SDK v2 when S3 credentials are configured.
	return fmt.Errorf("S3 storage not yet implemented")
}
