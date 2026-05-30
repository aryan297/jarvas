package entity

import (
	"time"

	"github.com/google/uuid"
)

type AgentType string

const (
	AgentTypeSupervisor AgentType = "SUPERVISOR"
	AgentTypeResearch   AgentType = "RESEARCH"
	AgentTypeCoding     AgentType = "CODING"
	AgentTypePlanning   AgentType = "PLANNING"
	AgentTypeWorkflow   AgentType = "WORKFLOW"
	AgentTypeCustom     AgentType = "CUSTOM"
)

// Agent is the aggregate root. It represents a reusable AI agent configuration.
// The actual Eino orchestration graph is built at runtime from this config.
type Agent struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Name          string
	Description   string
	Type          AgentType
	SystemPrompt  string
	Model         string
	Temperature   float64
	MaxTokens     int
	ToolsEnabled  []string
	MemoryEnabled bool
	RAGEnabled    bool
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewAgent(userID uuid.UUID, name string, agentType AgentType) *Agent {
	now := time.Now().UTC()
	return &Agent{
		ID:            uuid.New(),
		UserID:        userID,
		Name:          name,
		Type:          agentType,
		Model:         "gpt-4o",
		Temperature:   0.7,
		MaxTokens:     4096,
		MemoryEnabled: true,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
