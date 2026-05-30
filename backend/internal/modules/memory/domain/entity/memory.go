package entity

import (
	"time"

	"github.com/google/uuid"
)

type MemoryType string

const (
	MemoryFact         MemoryType = "FACT"
	MemoryPreference   MemoryType = "PREFERENCE"
	MemoryEvent        MemoryType = "EVENT"
	MemorySkill        MemoryType = "SKILL"
	MemoryRelationship MemoryType = "RELATIONSHIP"
)

// Memory is a single long-term memory entry.
// Short-term memory (last N messages) lives in Redis and is not modelled here.
// Semantic search vectors are in Qdrant; this entity owns the source of truth.
type Memory struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AgentID        *uuid.UUID
	Type           MemoryType
	Content        string
	Importance     float64 // 0.0 – 1.0
	QdrantID       *uuid.UUID
	LastAccessedAt *time.Time
	AccessCount    int
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewMemory(userID uuid.UUID, memType MemoryType, content string, importance float64) *Memory {
	now := time.Now().UTC()
	return &Memory{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       memType,
		Content:    content,
		Importance: importance,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (m *Memory) RecordAccess() {
	now := time.Now().UTC()
	m.LastAccessedAt = &now
	m.AccessCount++
}
