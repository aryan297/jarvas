package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/memory/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgMemoryRepository struct{ db *pgxpool.Pool }

func NewMemoryRepository(db *pgxpool.Pool) *pgMemoryRepository {
	return &pgMemoryRepository{db: db}
}

func (r *pgMemoryRepository) Save(ctx context.Context, m *entity.Memory) error {
	q := `INSERT INTO memories
	        (id, user_id, agent_id, type, content, importance, expires_at, created_at, updated_at)
	      VALUES ($1,$2,$3,$4::memory_type,$5,$6,$7,$8,$9)`
	_, err := r.db.Exec(ctx, q,
		m.ID, m.UserID, m.AgentID,
		string(m.Type), m.Content, m.Importance,
		m.ExpiresAt, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (r *pgMemoryRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Memory, int64, error) {
	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM memories WHERE user_id=$1`, userID,
	).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, agent_id, type::text, content, importance,
		       qdrant_id, last_accessed_at, access_count, expires_at, created_at, updated_at
		FROM memories
		WHERE user_id=$1
		ORDER BY importance DESC, created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*entity.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			continue
		}
		result = append(result, m)
	}
	return result, total, nil
}

func (r *pgMemoryRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Memory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, agent_id, type::text, content, importance,
		       qdrant_id, last_accessed_at, access_count, expires_at, created_at, updated_at
		FROM memories
		WHERE id=$1 AND user_id=$2`, id, userID)
	m, err := scanMemory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("memory")
		}
		return nil, err
	}
	return m, nil
}

func (r *pgMemoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM memories WHERE id=$1`, id)
	return err
}

func (r *pgMemoryRepository) UpdateQdrantID(ctx context.Context, id, qdrantID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE memories SET qdrant_id=$2, updated_at=NOW() WHERE id=$1`,
		id, qdrantID)
	return err
}

func (r *pgMemoryRepository) RecordAccess(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE memories
		SET last_accessed_at=NOW(), access_count=access_count+1, updated_at=NOW()
		WHERE id=$1`, id)
	return err
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanMemory(row scanner) (*entity.Memory, error) {
	var m entity.Memory
	var memType string
	var qdrantID *uuid.UUID
	var lastAccessed *time.Time
	var expiresAt *time.Time

	err := row.Scan(
		&m.ID, &m.UserID, &m.AgentID, &memType, &m.Content, &m.Importance,
		&qdrantID, &lastAccessed, &m.AccessCount, &expiresAt,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	m.Type = entity.MemoryType(memType)
	m.QdrantID = qdrantID
	m.LastAccessedAt = lastAccessed
	m.ExpiresAt = expiresAt
	return &m, nil
}
