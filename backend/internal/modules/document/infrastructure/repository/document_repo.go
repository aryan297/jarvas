package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/document/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgDocumentRepository struct{ db *pgxpool.Pool }

func NewDocumentRepository(db *pgxpool.Pool) *pgDocumentRepository {
	return &pgDocumentRepository{db: db}
}

func (r *pgDocumentRepository) Create(ctx context.Context, d *entity.Document) error {
	q := `INSERT INTO documents (id, user_id, name, type, mime_type, size_bytes, storage_key, status, created_at, updated_at)
	      VALUES ($1,$2,$3,$4::document_type,$5,$6,$7,$8::document_status,$9,$10)`
	_, err := r.db.Exec(ctx, q,
		d.ID, d.UserID, d.Name, string(d.Type), d.MimeType,
		d.SizeBytes, d.StorageKey, string(d.Status), d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *pgDocumentRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Document, error) {
	q := `SELECT id, user_id, name, type::text, mime_type, size_bytes, storage_key,
	             status::text, chunk_count, COALESCE(error_msg,''), created_at, updated_at
	      FROM documents WHERE id=$1 AND user_id=$2`
	return scanDoc(r.db.QueryRow(ctx, q, id, userID))
}

func (r *pgDocumentRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Document, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM documents WHERE user_id=$1`, userID).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, type::text, mime_type, size_bytes, storage_key,
		        status::text, chunk_count, COALESCE(error_msg,''), created_at, updated_at
		 FROM documents WHERE user_id=$1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docs []*entity.Document
	for rows.Next() {
		d, err := scanDoc(rows)
		if err == nil {
			docs = append(docs, d)
		}
	}
	return docs, total, nil
}

func (r *pgDocumentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.DocumentStatus, errMsg string) error {
	var msg interface{}
	if errMsg != "" {
		msg = errMsg
	}
	_, err := r.db.Exec(ctx,
		`UPDATE documents SET status=$2::document_status, error_msg=$3, updated_at=NOW() WHERE id=$1`,
		id, string(status), msg,
	)
	return err
}

func (r *pgDocumentRepository) UpdateIndexed(ctx context.Context, id uuid.UUID, chunkCount int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE documents SET status='INDEXED'::document_status, chunk_count=$2, updated_at=NOW() WHERE id=$1`,
		id, chunkCount,
	)
	return err
}

func (r *pgDocumentRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM documents WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// ── Scan helper ───────────────────────────────────────────────────────────────

type rowScanner interface{ Scan(...any) error }

func scanDoc(row rowScanner) (*entity.Document, error) {
	var d entity.Document
	var docType, status string
	err := row.Scan(
		&d.ID, &d.UserID, &d.Name, &docType, &d.MimeType,
		&d.SizeBytes, &d.StorageKey, &status, &d.ChunkCount, &d.ErrorMsg,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("document")
		}
		return nil, err
	}
	d.Type = entity.DocumentType(docType)
	d.Status = entity.DocumentStatus(status)
	return &d, nil
}
