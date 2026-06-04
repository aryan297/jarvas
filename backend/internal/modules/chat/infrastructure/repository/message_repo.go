package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/chat/domain/entity"
)

type pgMessageRepository struct{ db *pgxpool.Pool }

func NewMessageRepository(db *pgxpool.Pool) *pgMessageRepository {
	return &pgMessageRepository{db: db}
}

func (r *pgMessageRepository) Save(ctx context.Context, m *entity.Message) error {
	q := `INSERT INTO messages (id, conversation_id, role, content, token_count, model, created_at)
	      VALUES ($1,$2,$3::message_role,$4,$5,$6,$7)`
	_, err := r.db.Exec(ctx, q,
		m.ID, m.ConversationID, string(m.Role), m.Content,
		nullInt(m.TokenCount), nullStr(m.Model), m.CreatedAt,
	)
	return err
}

func (r *pgMessageRepository) FindByConversationID(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, conversation_id, role::text, content,
		        COALESCE(token_count, 0), COALESCE(model, ''), created_at
		 FROM messages
		 WHERE conversation_id=$1
		 ORDER BY created_at ASC
		 LIMIT $2 OFFSET $3`,
		convID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*entity.Message
	for rows.Next() {
		var m entity.Message
		var role string
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &role, &m.Content,
			&m.TokenCount, &m.Model, &m.CreatedAt,
		); err != nil {
			continue
		}
		m.Role = entity.MessageRole(role)
		msgs = append(msgs, &m)
	}
	return msgs, nil
}

func (r *pgMessageRepository) CountByConversationID(ctx context.Context, convID uuid.UUID) (int64, error) {
	var n int64
	r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages WHERE conversation_id=$1`, convID,
	).Scan(&n)
	return n, nil
}

func nullInt(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}
