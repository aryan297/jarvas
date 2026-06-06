package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/workflow/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgWorkflowRepository struct{ db *pgxpool.Pool }

func NewWorkflowRepository(db *pgxpool.Pool) *pgWorkflowRepository {
	return &pgWorkflowRepository{db: db}
}

func (r *pgWorkflowRepository) Save(ctx context.Context, w *entity.Workflow) error {
	def, _ := json.Marshal(w.Definition)
	_, err := r.db.Exec(ctx, `
		INSERT INTO workflows (id, user_id, name, description, status, definition,
		                       trigger_type, cron_expr, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5::workflow_status,$6,$7,$8,$9,$10)`,
		w.ID, w.UserID, w.Name, nullStr(w.Description),
		string(w.Status), def,
		nullStr(string(w.Trigger)), nullStr(w.CronExpr),
		w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (r *pgWorkflowRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Workflow, error) {
	q := `SELECT id, user_id, name, description, status::text, definition,
	             trigger_type, cron_expr, created_at, updated_at
	      FROM workflows WHERE id=$1 AND user_id=$2 AND status!='ARCHIVED'`
	row := r.db.QueryRow(ctx, q, id, userID)
	w, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("workflow")
		}
		return nil, err
	}
	return w, nil
}

func (r *pgWorkflowRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Workflow, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM workflows WHERE user_id=$1 AND status!='ARCHIVED'`, userID).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, description, status::text, definition,
		       trigger_type, cron_expr, created_at, updated_at
		FROM workflows WHERE user_id=$1 AND status!='ARCHIVED'
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*entity.Workflow
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err == nil {
			result = append(result, w)
		}
	}
	return result, total, nil
}

func (r *pgWorkflowRepository) FindActiveScheduled(ctx context.Context) ([]*entity.Workflow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, description, status::text, definition,
		       trigger_type, cron_expr, created_at, updated_at
		FROM workflows
		WHERE status='ACTIVE' AND trigger_type='SCHEDULE' AND cron_expr IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*entity.Workflow
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err == nil {
			result = append(result, w)
		}
	}
	return result, nil
}

func (r *pgWorkflowRepository) Update(ctx context.Context, w *entity.Workflow) error {
	def, _ := json.Marshal(w.Definition)
	_, err := r.db.Exec(ctx, `
		UPDATE workflows SET name=$2, description=$3, status=$4::workflow_status,
		  definition=$5, trigger_type=$6, cron_expr=$7, updated_at=$8
		WHERE id=$1`,
		w.ID, w.Name, nullStr(w.Description), string(w.Status), def,
		nullStr(string(w.Trigger)), nullStr(w.CronExpr), w.UpdatedAt,
	)
	return err
}

func (r *pgWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE workflows SET status='ARCHIVED', updated_at=$2 WHERE id=$1`,
		id, time.Now().UTC())
	return err
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanWorkflow(row scanner) (*entity.Workflow, error) {
	var w entity.Workflow
	var description, triggerType, cronExpr *string
	var statusStr string
	var defJSON []byte

	err := row.Scan(
		&w.ID, &w.UserID, &w.Name, &description, &statusStr, &defJSON,
		&triggerType, &cronExpr, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	w.Status = entity.WorkflowStatus(statusStr)
	if description != nil {
		w.Description = *description
	}
	if triggerType != nil {
		w.Trigger = entity.TriggerType(*triggerType)
	}
	if cronExpr != nil {
		w.CronExpr = *cronExpr
	}
	if len(defJSON) > 0 {
		_ = json.Unmarshal(defJSON, &w.Definition)
	}
	return &w, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
