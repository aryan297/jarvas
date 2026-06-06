package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	toolentity "github.com/jarvas/backend/internal/modules/tool/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// ── Tool repository ───────────────────────────────────────────────────────────

type pgToolRepository struct{ db *pgxpool.Pool }

func NewToolRepository(db *pgxpool.Pool) *pgToolRepository {
	return &pgToolRepository{db: db}
}

func (r *pgToolRepository) FindAll(ctx context.Context) ([]*toolentity.Tool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, display_name, description, category::text, schema, is_builtin, is_active, created_at
		FROM tools WHERE is_active=TRUE ORDER BY is_builtin DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*toolentity.Tool
	for rows.Next() {
		t, err := scanTool(rows)
		if err == nil {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *pgToolRepository) FindByName(ctx context.Context, name string) (*toolentity.Tool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, display_name, description, category::text, schema, is_builtin, is_active, created_at
		FROM tools WHERE name=$1 AND is_active=TRUE`, name)
	t, err := scanTool(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("tool")
		}
		return nil, err
	}
	return t, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanTool(row scanner) (*toolentity.Tool, error) {
	var t toolentity.Tool
	var category string
	var description *string
	var schemaJSON []byte

	err := row.Scan(&t.ID, &t.Name, &t.DisplayName, &description, &category,
		&schemaJSON, &t.IsBuiltin, &t.IsActive, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.Category = toolentity.ToolCategory(category)
	if description != nil {
		t.Description = *description
	}
	if len(schemaJSON) > 0 {
		_ = json.Unmarshal(schemaJSON, &t.Schema)
	}
	return &t, nil
}

// ── UserToolConfig repository ─────────────────────────────────────────────────

type pgUserToolConfigRepository struct{ db *pgxpool.Pool }

func NewUserToolConfigRepository(db *pgxpool.Pool) *pgUserToolConfigRepository {
	return &pgUserToolConfigRepository{db: db}
}

func (r *pgUserToolConfigRepository) Save(ctx context.Context, cfg *toolentity.UserToolConfig) error {
	data, _ := json.Marshal(cfg.Config)
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_tool_configs (id, user_id, tool_id, config, is_enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id, tool_id) DO UPDATE
		  SET config=$4, is_enabled=$5, updated_at=$7`,
		cfg.ID, cfg.UserID, cfg.ToolID, data, cfg.IsEnabled, now, now,
	)
	return err
}

func (r *pgUserToolConfigRepository) FindByUserAndTool(ctx context.Context, userID, toolID uuid.UUID) (*toolentity.UserToolConfig, error) {
	var cfg toolentity.UserToolConfig
	var configJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, tool_id, config, is_enabled, created_at, updated_at
		FROM user_tool_configs WHERE user_id=$1 AND tool_id=$2`,
		userID, toolID,
	).Scan(&cfg.ID, &cfg.UserID, &cfg.ToolID, &configJSON, &cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("tool config")
		}
		return nil, err
	}
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &cfg.Config)
	}
	return &cfg, nil
}

func (r *pgUserToolConfigRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*toolentity.UserToolConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, tool_id, config, is_enabled, created_at, updated_at
		FROM user_tool_configs WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*toolentity.UserToolConfig
	for rows.Next() {
		var cfg toolentity.UserToolConfig
		var configJSON []byte
		if err := rows.Scan(&cfg.ID, &cfg.UserID, &cfg.ToolID, &configJSON,
			&cfg.IsEnabled, &cfg.CreatedAt, &cfg.UpdatedAt); err == nil {
			if len(configJSON) > 0 {
				_ = json.Unmarshal(configJSON, &cfg.Config)
			}
			result = append(result, &cfg)
		}
	}
	return result, nil
}
