package entity

import (
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string
type MessageRole string

const (
	StatusActive   ConversationStatus = "ACTIVE"
	StatusArchived ConversationStatus = "ARCHIVED"
	StatusDeleted  ConversationStatus = "DELETED"

	RoleUser      MessageRole = "USER"
	RoleAssistant MessageRole = "ASSISTANT"
	RoleSystem    MessageRole = "SYSTEM"
	RoleTool      MessageRole = "TOOL"
)

type Conversation struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AgentID   *uuid.UUID
	Title     string
	Status    ConversationStatus
	Messages  []*Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	Role           MessageRole
	Content        string
	TokenCount     int
	Model          string
	ToolCalls      interface{}
	CreatedAt      time.Time
}

func NewConversation(userID uuid.UUID, agentID *uuid.UUID) *Conversation {
	now := time.Now().UTC()
	return &Conversation{
		ID:        uuid.New(),
		UserID:    userID,
		AgentID:   agentID,
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (c *Conversation) AddMessage(role MessageRole, content string) *Message {
	msg := &Message{
		ID:             uuid.New(),
		ConversationID: c.ID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now().UTC(),
	}
	c.Messages = append(c.Messages, msg)
	return msg
}
