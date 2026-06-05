package port

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/voice/domain/entity"
)

// SessionStore persists VoiceSessions — implemented with Redis (TTL 24h).
type SessionStore interface {
	Save(ctx context.Context, s *entity.VoiceSession) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.VoiceSession, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.VoiceSession, error)
}

// AudioStore uploads and retrieves raw audio files — implemented with MinIO.
type AudioStore interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) ([]byte, error)
}

// TranscriptionPort converts audio bytes to text — implemented with Whisper.
type TranscriptionPort interface {
	Transcribe(ctx context.Context, audioData []byte, filename, language string) (string, error)
}
