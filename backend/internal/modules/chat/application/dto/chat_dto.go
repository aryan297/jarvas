package dto

// SendMessageRequest is the payload for a chat turn.
type SendMessageRequest struct {
	ConversationID string `json:"conversation_id" validate:"omitempty,uuid"`
	AgentID        string `json:"agent_id"        validate:"omitempty,uuid"`
	Content        string `json:"content"         validate:"required,min=1,max=32000"`
	Stream         bool   `json:"stream"`
}

type CreateConversationRequest struct {
	AgentID string `json:"agent_id" validate:"omitempty,uuid"`
	Title   string `json:"title"    validate:"omitempty,max=500"`
}

type MessageResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Model     string `json:"model,omitempty"`
	CreatedAt string `json:"created_at"`
}

type ConversationResponse struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id,omitempty"`
	Title     string            `json:"title"`
	Status    string            `json:"status"`
	Messages  []MessageResponse `json:"messages,omitempty"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// StreamChunk is a single SSE frame for streaming responses.
type StreamChunk struct {
	Delta    string `json:"delta"`
	Done     bool   `json:"done"`
	TokensIn int    `json:"tokens_in,omitempty"`
	TokensOut int   `json:"tokens_out,omitempty"`
}
