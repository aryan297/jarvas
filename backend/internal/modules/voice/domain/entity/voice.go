package entity

import (
	"time"

	"github.com/google/uuid"
)

type TranscriptionStatus string

const (
	TranscriptionPending    TranscriptionStatus = "PENDING"
	TranscriptionProcessing TranscriptionStatus = "PROCESSING"
	TranscriptionCompleted  TranscriptionStatus = "COMPLETED"
	TranscriptionFailed     TranscriptionStatus = "FAILED"
)

// VoiceSession represents a voice interaction session.
// Audio is stored in MinIO; transcription flows through Whisper.
type VoiceSession struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	ConversationID   uuid.UUID
	AudioStorageKey  string
	DurationSeconds  float64
	Transcript       string
	Status           TranscriptionStatus
	LanguageCode     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewVoiceSession(userID, conversationID uuid.UUID, storageKey string) *VoiceSession {
	now := time.Now().UTC()
	return &VoiceSession{
		ID:              uuid.New(),
		UserID:          userID,
		ConversationID:  conversationID,
		AudioStorageKey: storageKey,
		Status:          TranscriptionPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
