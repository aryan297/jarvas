package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/document/application/dto"
	"github.com/jarvas/backend/internal/modules/document/application/port"
	"github.com/jarvas/backend/internal/modules/document/domain/entity"
	docevent "github.com/jarvas/backend/internal/modules/document/domain/event"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/eventbus"
)

// MaxFileSizeBytes — 50 MB
const MaxFileSizeBytes = 50 << 20

type DocumentService struct {
	docRepo   port.DocumentRepository
	chunkRepo port.ChunkRepository
	storage   port.StoragePort
	bus       *eventbus.Bus
}

func NewDocumentService(
	docRepo port.DocumentRepository,
	chunkRepo port.ChunkRepository,
	storage port.StoragePort,
	bus *eventbus.Bus,
) *DocumentService {
	return &DocumentService{docRepo: docRepo, chunkRepo: chunkRepo, storage: storage, bus: bus}
}

// Upload stores a file in MinIO, creates the DB record, and fires DocumentUploaded.
func (s *DocumentService) Upload(ctx context.Context, userID uuid.UUID, fileName, mimeType string, size int64, file io.Reader) (*dto.DocumentResponse, error) {
	if size > MaxFileSizeBytes {
		return nil, apperrors.BadRequest(fmt.Sprintf("file too large (max %d MB)", MaxFileSizeBytes>>20))
	}

	storageKey := fmt.Sprintf("documents/%s/%s/%s", userID, uuid.New(), fileName)
	if err := s.storage.Upload(ctx, storageKey, file, size, mimeType); err != nil {
		return nil, apperrors.Internal(fmt.Errorf("storage upload: %w", err))
	}

	doc := entity.NewDocument(userID, fileName, mimeType, storageKey, size)
	if err := s.docRepo.Create(ctx, doc); err != nil {
		_ = s.storage.Delete(ctx, storageKey) // rollback storage on DB failure
		return nil, apperrors.Internal(err)
	}

	// Fire async processing event.
	s.bus.Publish(ctx, docevent.DocumentUploaded{
		DocumentID: doc.ID,
		UserID:     userID,
		OccuredAt:  time.Now().UTC(),
	})

	return toDTO(doc), nil
}

// List returns paginated documents for a user.
func (s *DocumentService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.DocumentResponse, int64, error) {
	docs, total, err := s.docRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	var result []*dto.DocumentResponse
	for _, d := range docs {
		result = append(result, toDTO(d))
	}
	return result, total, nil
}

// GetByID returns a single document, verifying ownership.
func (s *DocumentService) GetByID(ctx context.Context, id, userID uuid.UUID) (*dto.DocumentResponse, error) {
	doc, err := s.docRepo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return toDTO(doc), nil
}

// Delete removes a document from storage, Postgres, and fires a cleanup event.
func (s *DocumentService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	doc, err := s.docRepo.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	// Delete chunks first (FK constraint).
	_ = s.chunkRepo.DeleteByDocumentID(ctx, id)

	// Delete from storage.
	_ = s.storage.Delete(ctx, doc.StorageKey)

	// Delete DB record.
	if err := s.docRepo.Delete(ctx, id, userID); err != nil {
		return apperrors.Internal(err)
	}

	// Publish — the RAG processor subscriber will delete Qdrant vectors.
	s.bus.Publish(ctx, docevent.DocumentDeleted{
		DocumentID: id,
		UserID:     userID,
		OccuredAt:  time.Now().UTC(),
	})

	return nil
}

// GetPresignedURL returns a temporary download URL from MinIO.
func (s *DocumentService) GetPresignedURL(ctx context.Context, id, userID uuid.UUID) (string, error) {
	doc, err := s.docRepo.FindByID(ctx, id, userID)
	if err != nil {
		return "", err
	}
	return s.storage.PresignedGetURL(ctx, doc.StorageKey)
}

// ListChunks returns raw text chunks for a document.
func (s *DocumentService) ListChunks(ctx context.Context, id, userID uuid.UUID) (interface{}, error) {
	if _, err := s.docRepo.FindByID(ctx, id, userID); err != nil {
		return nil, err
	}
	return s.chunkRepo.FindByDocumentID(ctx, id)
}

// ── DTO mapper ────────────────────────────────────────────────────────────────

func toDTO(d *entity.Document) *dto.DocumentResponse {
	return &dto.DocumentResponse{
		ID:         d.ID.String(),
		Name:       d.Name,
		Type:       string(d.Type),
		MimeType:   d.MimeType,
		SizeBytes:  d.SizeBytes,
		Status:     string(d.Status),
		ChunkCount: d.ChunkCount,
		CreatedAt:  d.CreatedAt.Format(time.RFC3339),
	}
}
