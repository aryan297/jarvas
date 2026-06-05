package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/jarvas/backend/internal/shared/config"
)

type MinIOAudioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOAudioStorage(cfg config.MinIOConfig) (*MinIOAudioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio audio storage: %w", err)
	}
	return &MinIOAudioStorage{client: client, bucket: cfg.BucketAudio}, nil
}

func (s *MinIOAudioStorage) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("audio upload %q: %w", key, err)
	}
	return nil
}

func (s *MinIOAudioStorage) Download(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("audio download %q: %w", key, err)
	}
	defer obj.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, obj); err != nil {
		return nil, fmt.Errorf("audio read %q: %w", key, err)
	}
	return buf.Bytes(), nil
}
