package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/workflow/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgRunRepository struct{ db *pgxpool.Pool }

func NewRunRepository(db *pgxpool.Pool) *pgRunRepository {
	return &pgRunRepository{db: db}
}

func (r *pgRunRepository) Save(ctx context.Context, run *entity.WorkflowRun) error {
	payload, _ := json.Marshal(run.TriggerPayload)
	_, err := r.db.Exec(ctx, `
		INSERT INTO workflow_runs
		  (id, workflow_id, user_id, status, trigger_payload, created_at)
		VALUES ($1,$2,$3,$4::run_status,$5,$6)`,
		run.ID, run.WorkflowID, run.UserID, string(run.Status), payload, run.CreatedAt,
	)
	return err
}

func (r *pgRunRepository) Update(ctx context.Context, run *entity.WorkflowRun) error {
	result, _ := json.Marshal(run.Result)
	_, err := r.db.Exec(ctx, `
		UPDATE workflow_runs SET
		  status=$2::run_status, result=$3, error_msg=$4,
		  started_at=$5, completed_at=$6
		WHERE id=$1`,
		run.ID, string(run.Status), result,
		nullStr(run.ErrorMsg), run.StartedAt, run.CompletedAt,
	)
	return err
}

func (r *pgRunRepository) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*entity.WorkflowRun, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM workflow_runs WHERE workflow_id=$1`, workflowID).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT id, workflow_id, user_id, status::text, result, error_msg,
		       started_at, completed_at, created_at
		FROM workflow_runs WHERE workflow_id=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		workflowID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*entity.WorkflowRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err == nil {
			result = append(result, run)
		}
	}
	return result, total, nil
}

func (r *pgRunRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.WorkflowRun, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, workflow_id, user_id, status::text, result, error_msg,
		       started_at, completed_at, created_at
		FROM workflow_runs WHERE id=$1`, id)
	run, err := scanRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("workflow run")
		}
		return nil, err
	}
	return run, nil
}

func scanRun(row scanner) (*entity.WorkflowRun, error) {
	var run entity.WorkflowRun
	var statusStr string
	var resultJSON []byte
	var errMsg *string

	err := row.Scan(
		&run.ID, &run.WorkflowID, &run.UserID, &statusStr, &resultJSON, &errMsg,
		&run.StartedAt, &run.CompletedAt, &run.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	run.Status = entity.RunStatus(statusStr)
	if errMsg != nil {
		run.ErrorMsg = *errMsg
	}
	if len(resultJSON) > 0 {
		_ = json.Unmarshal(resultJSON, &run.Result)
	}
	return &run, nil
}
