package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/jarvas/backend/internal/shared/config"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(cfg config.MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, _ := client.BucketExists(ctx, cfg.BucketDocuments)
	if !exists {
		if err := client.MakeBucket(ctx, cfg.BucketDocuments, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
		logger.Info("minio bucket created", zap.String("bucket", cfg.BucketDocuments))
	}

	logger.Info("minio connected", zap.String("endpoint", cfg.Endpoint))
	return &MinIOStorage{client: client, bucket: cfg.BucketDocuments}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIOStorage) PresignedGetURL(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, 24*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
