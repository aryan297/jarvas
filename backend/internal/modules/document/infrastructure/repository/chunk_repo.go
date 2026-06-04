package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/document/domain/entity"
)

type pgChunkRepository struct{ db *pgxpool.Pool }

func NewChunkRepository(db *pgxpool.Pool) *pgChunkRepository {
	return &pgChunkRepository{db: db}
}

func (r *pgChunkRepository) SaveBatch(ctx context.Context, chunks []*entity.DocumentChunk) error {
	// Use a single transaction for atomicity.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, c := range chunks {
		_, err := tx.Exec(ctx,
			`INSERT INTO document_chunks (id, document_id, user_id, content, chunk_index, token_count, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			c.ID, c.DocumentID, c.UserID, c.Content, c.ChunkIndex, c.TokenCount, c.CreatedAt,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *pgChunkRepository) FindByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*entity.DocumentChunk, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, document_id, user_id, content, chunk_index, COALESCE(token_count,0), created_at
		 FROM document_chunks WHERE document_id=$1 ORDER BY chunk_index ASC`,
		documentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*entity.DocumentChunk
	for rows.Next() {
		var c entity.DocumentChunk
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.UserID, &c.Content, &c.ChunkIndex, &c.TokenCount, &c.CreatedAt); err == nil {
			chunks = append(chunks, &c)
		}
	}
	return chunks, nil
}

func (r *pgChunkRepository) DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM document_chunks WHERE document_id=$1`, documentID)
	return err
}
