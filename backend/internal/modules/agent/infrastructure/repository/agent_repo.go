package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/agent/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgAgentRepository struct{ db *pgxpool.Pool }

func NewAgentRepository(db *pgxpool.Pool) *pgAgentRepository {
	return &pgAgentRepository{db: db}
}

func (r *pgAgentRepository) Save(ctx context.Context, a *entity.Agent) error {
	tools, _ := json.Marshal(a.ToolsEnabled)
	q := `INSERT INTO agents
	        (id, user_id, name, description, type, system_prompt, model,
	         temperature, max_tokens, tools_enabled, memory_enabled, rag_enabled,
	         is_active, created_at, updated_at)
	      VALUES ($1,$2,$3,$4,$5::agent_type,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`
	_, err := r.db.Exec(ctx, q,
		a.ID, a.UserID, a.Name, nullStr(a.Description),
		string(a.Type), nullStr(a.SystemPrompt), a.Model,
		a.Temperature, a.MaxTokens, tools,
		a.MemoryEnabled, a.RAGEnabled, a.IsActive,
		a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *pgAgentRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Agent, error) {
	q := `SELECT id, user_id, name, description, type::text, system_prompt, model,
	             temperature, max_tokens, tools_enabled, memory_enabled, rag_enabled,
	             is_active, created_at, updated_at
	      FROM agents WHERE id=$1`

	// If userID is provided (non-nil), also filter by user_id for ownership check.
	var row pgx.Row
	if userID != uuid.Nil {
		q += " AND user_id=$2"
		row = r.db.QueryRow(ctx, q, id, userID)
	} else {
		row = r.db.QueryRow(ctx, q, id)
	}

	a, err := scanAgent(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("agent")
		}
		return nil, err
	}
	return a, nil
}

func (r *pgAgentRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Agent, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE user_id=$1 AND is_active=TRUE`, userID).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, description, type::text, system_prompt, model,
		       temperature, max_tokens, tools_enabled, memory_enabled, rag_enabled,
		       is_active, created_at, updated_at
		FROM agents WHERE user_id=$1 AND is_active=TRUE
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*entity.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err == nil {
			result = append(result, a)
		}
	}
	return result, total, nil
}

func (r *pgAgentRepository) Update(ctx context.Context, a *entity.Agent) error {
	tools, _ := json.Marshal(a.ToolsEnabled)
	_, err := r.db.Exec(ctx, `
		UPDATE agents SET
		  name=$2, description=$3, system_prompt=$4, model=$5,
		  temperature=$6, max_tokens=$7, tools_enabled=$8,
		  memory_enabled=$9, rag_enabled=$10, updated_at=$11
		WHERE id=$1`,
		a.ID, a.Name, nullStr(a.Description), nullStr(a.SystemPrompt),
		a.Model, a.Temperature, a.MaxTokens, tools,
		a.MemoryEnabled, a.RAGEnabled, a.UpdatedAt,
	)
	return err
}

func (r *pgAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE agents SET is_active=FALSE, updated_at=$2 WHERE id=$1`,
		id, time.Now().UTC())
	return err
}

// ── Scan helper ───────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(row scanner) (*entity.Agent, error) {
	var a entity.Agent
	var agentType string
	var description, systemPrompt *string
	var toolsJSON []byte

	err := row.Scan(
		&a.ID, &a.UserID, &a.Name, &description, &agentType, &systemPrompt,
		&a.Model, &a.Temperature, &a.MaxTokens, &toolsJSON,
		&a.MemoryEnabled, &a.RAGEnabled, &a.IsActive,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	a.Type = entity.AgentType(agentType)
	if description != nil {
		a.Description = *description
	}
	if systemPrompt != nil {
		a.SystemPrompt = *systemPrompt
	}
	if len(toolsJSON) > 0 {
		_ = json.Unmarshal(toolsJSON, &a.ToolsEnabled)
	}
	if a.ToolsEnabled == nil {
		a.ToolsEnabled = []string{}
	}
	return &a, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
