package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/voice/domain/entity"
	sharedcache "github.com/jarvas/backend/internal/shared/cache"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

const sessionTTL = 24 * time.Hour

type RedisSessionStore struct {
	cache *sharedcache.Client
}

func NewRedisSessionStore(c *sharedcache.Client) *RedisSessionStore {
	return &RedisSessionStore{cache: c}
}

func (s *RedisSessionStore) Save(ctx context.Context, session *entity.VoiceSession) error {
	if err := s.cache.Set(ctx, sessionKey(session.ID), session, sessionTTL); err != nil {
		return fmt.Errorf("session save: %w", err)
	}

	// Maintain an ordered list of the user's last 20 session IDs.
	var ids []string
	_ = s.cache.Get(ctx, userListKey(session.UserID), &ids)
	// Prepend newest; deduplicate.
	newIDs := []string{session.ID.String()}
	for _, id := range ids {
		if id != session.ID.String() {
			newIDs = append(newIDs, id)
		}
	}
	if len(newIDs) > 20 {
		newIDs = newIDs[:20]
	}
	_ = s.cache.Set(ctx, userListKey(session.UserID), newIDs, sessionTTL)
	return nil
}

func (s *RedisSessionStore) FindByID(ctx context.Context, id uuid.UUID) (*entity.VoiceSession, error) {
	var session entity.VoiceSession
	if err := s.cache.Get(ctx, sessionKey(id), &session); err != nil {
		if sharedcache.IsNil(err) {
			return nil, apperrors.NotFound("voice session")
		}
		return nil, fmt.Errorf("session get: %w", err)
	}
	return &session, nil
}

func (s *RedisSessionStore) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.VoiceSession, error) {
	var ids []string
	if err := s.cache.Get(ctx, userListKey(userID), &ids); err != nil {
		return []*entity.VoiceSession{}, nil // no sessions yet — not an error
	}

	sessions := make([]*entity.VoiceSession, 0, len(ids))
	for _, idStr := range ids {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		session, err := s.FindByID(ctx, id)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func sessionKey(id uuid.UUID) string  { return fmt.Sprintf("voice:session:%s", id) }
func userListKey(id uuid.UUID) string { return fmt.Sprintf("voice:user:%s", id) }
