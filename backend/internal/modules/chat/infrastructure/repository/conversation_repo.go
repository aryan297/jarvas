package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/chat/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgConversationRepository struct{ db *pgxpool.Pool }

func NewConversationRepository(db *pgxpool.Pool) *pgConversationRepository {
	return &pgConversationRepository{db: db}
}

func (r *pgConversationRepository) Create(ctx context.Context, c *entity.Conversation) error {
	q := `INSERT INTO conversations (id, user_id, agent_id, title, status, created_at, updated_at)
	      VALUES ($1,$2,$3,$4,$5::conversation_status,$6,$7)`
	_, err := r.db.Exec(ctx, q,
		c.ID, c.UserID, c.AgentID, nullStr(c.Title),
		string(c.Status), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *pgConversationRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Conversation, error) {
	q := `SELECT id, user_id, agent_id, title, status::text, created_at, updated_at
	      FROM conversations
	      WHERE id=$1 AND user_id=$2 AND status!='DELETED'`
	return scanConv(r.db.QueryRow(ctx, q, id, userID))
}

func (r *pgConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Conversation, int64, error) {
	var total int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM conversations WHERE user_id=$1 AND status='ACTIVE'`, userID,
	).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, agent_id, title, status::text, created_at, updated_at
		 FROM conversations
		 WHERE user_id=$1 AND status='ACTIVE'
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var convs []*entity.Conversation
	for rows.Next() {
		c, err := scanConv(rows)
		if err != nil {
			continue
		}
		convs = append(convs, c)
	}
	return convs, total, nil
}

func (r *pgConversationRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE conversations SET title=$2, updated_at=NOW() WHERE id=$1`, id, title,
	)
	return err
}

func (r *pgConversationRepository) Archive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE conversations SET status='ARCHIVED', updated_at=NOW() WHERE id=$1`, id,
	)
	return err
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConv(row rowScanner) (*entity.Conversation, error) {
	var c entity.Conversation
	var agentID *uuid.UUID
	var title *string
	var status string
	err := row.Scan(&c.ID, &c.UserID, &agentID, &title, &status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("conversation")
		}
		return nil, err
	}
	c.AgentID = agentID
	c.Status = entity.ConversationStatus(status)
	if title != nil {
		c.Title = *title
	}
	return &c, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
