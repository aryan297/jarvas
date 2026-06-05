package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/voice/application/dto"
	"github.com/jarvas/backend/internal/modules/voice/application/port"
	"github.com/jarvas/backend/internal/modules/voice/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// maxAudioBytes = 25 MB (Whisper API limit)
const maxAudioBytes = 25 * 1024 * 1024

// Accepted audio MIME types.
var allowedMIME = map[string]bool{
	"audio/webm":  true,
	"audio/mp4":   true,
	"audio/mpeg":  true,
	"audio/ogg":   true,
	"audio/wav":   true,
	"audio/x-wav": true,
	"audio/m4a":   true,
}

type VoiceService struct {
	sessions   port.SessionStore
	audioStore port.AudioStore
	transcribe port.TranscriptionPort
}

func NewVoiceService(
	sessions port.SessionStore,
	audioStore port.AudioStore,
	transcribe port.TranscriptionPort,
) *VoiceService {
	return &VoiceService{
		sessions:   sessions,
		audioStore: audioStore,
		transcribe: transcribe,
	}
}

// Upload stores the audio file and kicks off async transcription.
// Returns the session ID so the client can poll for the result.
func (s *VoiceService) Upload(
	ctx context.Context,
	userID uuid.UUID,
	conversationID string,
	language string,
	file multipart.File,
	header *multipart.FileHeader,
) (*dto.UploadVoiceResponse, error) {
	if header.Size > maxAudioBytes {
		return nil, apperrors.BadRequest("audio file exceeds 25 MB limit")
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/webm"
	}
	if !allowedMIME[strings.ToLower(contentType)] {
		return nil, apperrors.BadRequest("unsupported audio format")
	}

	convID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid conversation_id")
	}

	// Generate storage key: audio/{user_id}/{session_id}{ext}
	sessionID := uuid.New()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".webm"
	}
	storageKey := fmt.Sprintf("audio/%s/%s%s", userID, sessionID, ext)

	// Upload audio to MinIO.
	if err := s.audioStore.Upload(ctx, storageKey, file, header.Size, contentType); err != nil {
		return nil, apperrors.Internal(fmt.Errorf("audio upload: %w", err))
	}

	// Create session record (PENDING).
	session := entity.NewVoiceSession(userID, convID, storageKey)
	session.ID = sessionID
	if language != "" {
		session.LanguageCode = language
	}
	if err := s.sessions.Save(ctx, session); err != nil {
		return nil, apperrors.Internal(err)
	}

	// Fire-and-forget transcription in background.
	filename := fmt.Sprintf("audio%s", ext)
	go s.runTranscription(context.Background(), session, filename)

	return &dto.UploadVoiceResponse{
		SessionID: session.ID.String(),
		Status:    string(session.Status),
	}, nil
}

// GetSession returns the current state of a voice session (used for polling).
func (s *VoiceService) GetSession(ctx context.Context, sessionID, userID uuid.UUID) (*dto.VoiceSessionResponse, error) {
	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, apperrors.NotFound("voice session")
	}
	return toDTO(session), nil
}

// ListSessions returns a user's recent voice sessions.
func (s *VoiceService) ListSessions(ctx context.Context, userID uuid.UUID) ([]*dto.VoiceSessionResponse, error) {
	sessions, err := s.sessions.ListByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	result := make([]*dto.VoiceSessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, toDTO(sess))
	}
	return result, nil
}

// ── Background transcription ──────────────────────────────────────────────────

func (s *VoiceService) runTranscription(ctx context.Context, session *entity.VoiceSession, filename string) {
	// Update status to PROCESSING.
	session.Status = entity.TranscriptionProcessing
	session.UpdatedAt = time.Now().UTC()
	_ = s.sessions.Save(ctx, session)

	// Download audio from MinIO.
	audioData, err := s.audioStore.Download(ctx, session.AudioStorageKey)
	if err != nil {
		s.failSession(ctx, session, fmt.Sprintf("download error: %v", err))
		return
	}

	// Transcribe via Whisper.
	transcript, err := s.transcribe.Transcribe(ctx, audioData, filename, session.LanguageCode)
	if err != nil {
		s.failSession(ctx, session, fmt.Sprintf("transcription error: %v", err))
		return
	}

	// Mark completed.
	session.Status = entity.TranscriptionCompleted
	session.Transcript = transcript
	session.UpdatedAt = time.Now().UTC()
	_ = s.sessions.Save(ctx, session)
}

func (s *VoiceService) failSession(ctx context.Context, session *entity.VoiceSession, reason string) {
	session.Status = entity.TranscriptionFailed
	session.Transcript = reason
	session.UpdatedAt = time.Now().UTC()
	_ = s.sessions.Save(ctx, session)
}

// ── DTO mapper ────────────────────────────────────────────────────────────────

func toDTO(s *entity.VoiceSession) *dto.VoiceSessionResponse {
	return &dto.VoiceSessionResponse{
		ID:             s.ID.String(),
		ConversationID: s.ConversationID.String(),
		Status:         string(s.Status),
		Transcript:     s.Transcript,
		DurationSecs:   s.DurationSeconds,
		LanguageCode:   s.LanguageCode,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
	}
}
