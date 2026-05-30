package entity

import (
	"time"

	"github.com/google/uuid"
)

type ToolCategory string

const (
	CategoryDatabase      ToolCategory = "DATABASE"
	CategoryHTTP          ToolCategory = "HTTP"
	CategoryProductivity  ToolCategory = "PRODUCTIVITY"
	CategoryCommunication ToolCategory = "COMMUNICATION"
	CategoryDevelopment   ToolCategory = "DEVELOPMENT"
	CategoryCustom        ToolCategory = "CUSTOM"
)

// Tool is the system-level tool definition. It is registered globally.
// Pluggable: new tools implement the Executor interface.
type Tool struct {
	ID          uuid.UUID
	Name        string
	DisplayName string
	Description string
	Category    ToolCategory
	Schema      map[string]interface{} // OpenAPI-style input schema
	IsBuiltin   bool
	IsActive    bool
	CreatedAt   time.Time
}

// UserToolConfig holds per-user credentials/settings for a tool.
type UserToolConfig struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ToolID    uuid.UUID
	Config    map[string]interface{} // encrypted at rest
	IsEnabled bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Executor is the interface every tool must implement.
// The tool registry maps tool names to executors.
type Executor interface {
	Name() string
	Execute(ctx interface{}, input map[string]interface{}) (interface{}, error)
}
