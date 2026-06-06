package dto

import "github.com/jarvas/backend/internal/modules/workflow/domain/entity"

type CreateWorkflowRequest struct {
	Name        string                     `json:"name"        binding:"required,min=1,max=255"`
	Description string                     `json:"description"`
	Definition  entity.WorkflowDefinition  `json:"definition"`
	TriggerType string                     `json:"trigger_type"`
	CronExpr    string                     `json:"cron_expr"`
}

type UpdateWorkflowRequest struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Status      string                     `json:"status"`
	Definition  *entity.WorkflowDefinition `json:"definition"`
	TriggerType string                     `json:"trigger_type"`
	CronExpr    string                     `json:"cron_expr"`
}

type TriggerRunRequest struct {
	Payload map[string]interface{} `json:"payload"`
}

type WorkflowResponse struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Status      string                    `json:"status"`
	Definition  entity.WorkflowDefinition `json:"definition"`
	TriggerType string                    `json:"trigger_type,omitempty"`
	CronExpr    string                    `json:"cron_expr,omitempty"`
	CreatedAt   string                    `json:"created_at"`
	UpdatedAt   string                    `json:"updated_at"`
}

type RunResponse struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	StartedAt   string                 `json:"started_at,omitempty"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}
