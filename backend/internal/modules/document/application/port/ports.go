package port

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/document/domain/entity"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *entity.Document) error
	FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Document, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Document, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.DocumentStatus, errMsg string) error
	UpdateIndexed(ctx context.Context, id uuid.UUID, chunkCount int) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type ChunkRepository interface {
	SaveBatch(ctx context.Context, chunks []*entity.DocumentChunk) error
	FindByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*entity.DocumentChunk, error)
	DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
}

// StoragePort abstracts the object store (MinIO in production).
type StoragePort interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignedGetURL(ctx context.Context, key string) (string, error)
}
