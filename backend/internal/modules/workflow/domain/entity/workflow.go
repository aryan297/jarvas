package entity

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowStatus string
type RunStatus      string
type TriggerType    string

const (
	WorkflowDraft    WorkflowStatus = "DRAFT"
	WorkflowActive   WorkflowStatus = "ACTIVE"
	WorkflowPaused   WorkflowStatus = "PAUSED"
	WorkflowArchived WorkflowStatus = "ARCHIVED"

	RunPending   RunStatus = "PENDING"
	RunRunning   RunStatus = "RUNNING"
	RunCompleted RunStatus = "COMPLETED"
	RunFailed    RunStatus = "FAILED"
	RunCancelled RunStatus = "CANCELLED"

	TriggerManual   TriggerType = "MANUAL"
	TriggerSchedule TriggerType = "SCHEDULE"
	TriggerWebhook  TriggerType = "WEBHOOK"
	TriggerEvent    TriggerType = "EVENT"
)

// WorkflowDefinition is the DAG stored as JSON in the DB.
type WorkflowDefinition struct {
	Nodes   []WorkflowNode `json:"nodes"`
	Edges   []WorkflowEdge `json:"edges"`
	Trigger TriggerConfig  `json:"trigger"`
}

type WorkflowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "agent", "tool", "condition", "delay"
	Config   map[string]interface{} `json:"config"`
}

type WorkflowEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

type TriggerConfig struct {
	Type     TriggerType `json:"type"`
	CronExpr string      `json:"cron_expr,omitempty"`
	EventName string     `json:"event_name,omitempty"`
}

type Workflow struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description string
	Status      WorkflowStatus
	Definition  WorkflowDefinition
	Trigger     TriggerType
	CronExpr    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkflowRun struct {
	ID             uuid.UUID
	WorkflowID     uuid.UUID
	UserID         uuid.UUID
	Status         RunStatus
	TriggerPayload map[string]interface{}
	Result         map[string]interface{}
	ErrorMsg       string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
}
