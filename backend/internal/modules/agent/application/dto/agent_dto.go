package dto

type CreateAgentRequest struct {
	Name          string   `json:"name"           validate:"required,min=1,max=100"`
	Description   string   `json:"description"    validate:"omitempty,max=500"`
	Type          string   `json:"type"           validate:"required,oneof=SUPERVISOR RESEARCH CODING PLANNING WORKFLOW CUSTOM"`
	SystemPrompt  string   `json:"system_prompt"  validate:"omitempty,max=8000"`
	Model         string   `json:"model"          validate:"omitempty"`
	Temperature   float64  `json:"temperature"    validate:"omitempty,min=0,max=2"`
	MaxTokens     int      `json:"max_tokens"     validate:"omitempty,min=1,max=128000"`
	ToolsEnabled  []string `json:"tools_enabled"`
	MemoryEnabled bool     `json:"memory_enabled"`
	RAGEnabled    bool     `json:"rag_enabled"`
}

type UpdateAgentRequest struct {
	Name          string   `json:"name"           validate:"omitempty,min=1,max=100"`
	Description   string   `json:"description"    validate:"omitempty,max=500"`
	SystemPrompt  string   `json:"system_prompt"  validate:"omitempty,max=8000"`
	Model         string   `json:"model"`
	Temperature   float64  `json:"temperature"`
	MaxTokens     int      `json:"max_tokens"`
	ToolsEnabled  []string `json:"tools_enabled"`
	MemoryEnabled bool     `json:"memory_enabled"`
	RAGEnabled    bool     `json:"rag_enabled"`
}

type AgentResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Type          string   `json:"type"`
	Model         string   `json:"model"`
	Temperature   float64  `json:"temperature"`
	MaxTokens     int      `json:"max_tokens"`
	ToolsEnabled  []string `json:"tools_enabled"`
	MemoryEnabled bool     `json:"memory_enabled"`
	RAGEnabled    bool     `json:"rag_enabled"`
	IsActive      bool     `json:"is_active"`
	CreatedAt     string   `json:"created_at"`
}
