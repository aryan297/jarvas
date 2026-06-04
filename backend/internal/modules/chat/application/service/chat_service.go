package service

import (
	"context"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/chat/application/dto"
	"github.com/jarvas/backend/internal/modules/chat/application/port"
	"github.com/jarvas/backend/internal/modules/chat/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/cache"
	"github.com/jarvas/backend/internal/shared/eventbus"
)

const (
	shortTermTTL   = 2 * time.Hour
	shortTermLimit = 20
)

type ChatConfig struct {
	OpenAIKey string
	Model     string
	MaxTokens int
}

type ChatService struct {
	convRepo port.ConversationRepository
	msgRepo  port.MessageRepository
	cache    *cache.Client
	bus      *eventbus.Bus
	oai      *openai.Client
	cfg      ChatConfig
}

func NewChatService(
	convRepo port.ConversationRepository,
	msgRepo port.MessageRepository,
	cache *cache.Client,
	bus *eventbus.Bus,
	cfg ChatConfig,
) *ChatService {
	return &ChatService{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		cache:    cache,
		bus:      bus,
		oai:      openai.NewClient(cfg.OpenAIKey),
		cfg:      cfg,
	}
}

// CreateConversation creates a new conversation and returns its DTO.
func (s *ChatService) CreateConversation(ctx context.Context, userID uuid.UUID, req dto.CreateConversationRequest) (*dto.ConversationResponse, error) {
	var agentID *uuid.UUID
	if req.AgentID != "" {
		id, err := uuid.Parse(req.AgentID)
		if err != nil {
			return nil, apperrors.BadRequest("invalid agent_id")
		}
		agentID = &id
	}
	conv := entity.NewConversation(userID, agentID)
	conv.Title = req.Title
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toConvDTO(conv), nil
}

// ListConversations returns paginated conversations for a user.
func (s *ChatService) ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.ConversationResponse, int64, error) {
	convs, total, err := s.convRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	var result []*dto.ConversationResponse
	for _, c := range convs {
		result = append(result, toConvDTO(c))
	}
	return result, total, nil
}

// GetConversation returns a single conversation with its recent messages.
func (s *ChatService) GetConversation(ctx context.Context, convID, userID uuid.UUID) (*dto.ConversationResponse, error) {
	conv, err := s.convRepo.FindByID(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	msgs, _ := s.msgRepo.FindByConversationID(ctx, convID, 50, 0)
	resp := toConvDTO(conv)
	for _, m := range msgs {
		resp.Messages = append(resp.Messages, *toMsgDTO(m))
	}
	return resp, nil
}

// ArchiveConversation soft-deletes a conversation.
func (s *ChatService) ArchiveConversation(ctx context.Context, convID, userID uuid.UUID) error {
	// Verify ownership first.
	if _, err := s.convRepo.FindByID(ctx, convID, userID); err != nil {
		return err
	}
	return s.convRepo.Archive(ctx, convID)
}

// ListMessages returns paginated messages for a conversation.
func (s *ChatService) ListMessages(ctx context.Context, convID, userID uuid.UUID, limit, offset int) ([]*dto.MessageResponse, int64, error) {
	// Verify ownership.
	if _, err := s.convRepo.FindByID(ctx, convID, userID); err != nil {
		return nil, 0, err
	}
	total, _ := s.msgRepo.CountByConversationID(ctx, convID)
	msgs, err := s.msgRepo.FindByConversationID(ctx, convID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	var result []*dto.MessageResponse
	for _, m := range msgs {
		result = append(result, toMsgDTO(m))
	}
	return result, total, nil
}

// SendMessage sends a user message, calls OpenAI, and returns the assistant reply.
func (s *ChatService) SendMessage(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (*dto.MessageResponse, error) {
	conv, err := s.resolveConversation(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	userMsg := s.saveMessage(ctx, conv.ID, entity.RoleUser, req.Content, "", 0)

	history := s.loadHistory(ctx, conv.ID)
	oaiMessages := buildOAIMessages(history, req.Content)

	resp, err := s.oai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     s.cfg.Model,
		Messages:  oaiMessages,
		MaxTokens: s.cfg.MaxTokens,
	})
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("openai: %w", err))
	}

	content := resp.Choices[0].Message.Content
	asstMsg := s.saveMessage(ctx, conv.ID, entity.RoleAssistant, content, s.cfg.Model, resp.Usage.TotalTokens)

	s.appendHistory(ctx, conv.ID, userMsg, asstMsg)
	return toMsgDTO(asstMsg), nil
}

// StreamMessage sends a user message and returns a channel of delta strings for SSE.
func (s *ChatService) StreamMessage(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (<-chan string, error) {
	conv, err := s.resolveConversation(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	userMsg := s.saveMessage(ctx, conv.ID, entity.RoleUser, req.Content, "", 0)
	history := s.loadHistory(ctx, conv.ID)
	oaiMessages := buildOAIMessages(history, req.Content)

	stream, err := s.oai.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     s.cfg.Model,
		Messages:  oaiMessages,
		MaxTokens: s.cfg.MaxTokens,
		Stream:    true,
	})
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("openai stream: %w", err))
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer stream.Close()

		var full string
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			delta := chunk.Choices[0].Delta.Content
			if delta == "" {
				continue
			}
			full += delta
			ch <- delta
		}

		asstMsg := s.saveMessage(ctx, conv.ID, entity.RoleAssistant, full, s.cfg.Model, 0)
		s.appendHistory(ctx, conv.ID, userMsg, asstMsg)
	}()

	return ch, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

// resolveConversation gets an existing conversation or creates one on-the-fly.
func (s *ChatService) resolveConversation(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (*entity.Conversation, error) {
	if req.ConversationID != "" {
		convID, err := uuid.Parse(req.ConversationID)
		if err != nil {
			return nil, apperrors.BadRequest("invalid conversation_id")
		}
		return s.convRepo.FindByID(ctx, convID, userID)
	}
	// Auto-create a conversation when none is specified.
	var agentID *uuid.UUID
	if req.AgentID != "" {
		id, _ := uuid.Parse(req.AgentID)
		agentID = &id
	}
	conv := entity.NewConversation(userID, agentID)
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, apperrors.Internal(err)
	}
	return conv, nil
}

func (s *ChatService) saveMessage(ctx context.Context, convID uuid.UUID, role entity.MessageRole, content, model string, tokens int) *entity.Message {
	m := &entity.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
		Model:          model,
		TokenCount:     tokens,
		CreatedAt:      time.Now().UTC(),
	}
	_ = s.msgRepo.Save(ctx, m)
	return m
}

func (s *ChatService) loadHistory(ctx context.Context, convID uuid.UUID) []entity.Message {
	key := fmt.Sprintf("short_term:%s", convID)
	var msgs []entity.Message
	if err := s.cache.Get(ctx, key, &msgs); err == nil {
		return msgs
	}
	dbMsgs, _ := s.msgRepo.FindByConversationID(ctx, convID, shortTermLimit, 0)
	for _, m := range dbMsgs {
		msgs = append(msgs, *m)
	}
	_ = s.cache.Set(ctx, key, msgs, shortTermTTL)
	return msgs
}

func (s *ChatService) appendHistory(ctx context.Context, convID uuid.UUID, msgs ...*entity.Message) {
	key := fmt.Sprintf("short_term:%s", convID)
	var history []entity.Message
	_ = s.cache.Get(ctx, key, &history)
	for _, m := range msgs {
		history = append(history, *m)
	}
	if len(history) > shortTermLimit {
		history = history[len(history)-shortTermLimit:]
	}
	_ = s.cache.Set(ctx, key, history, shortTermTTL)
}

func buildOAIMessages(history []entity.Message, userMsg string) []openai.ChatCompletionMessage {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: "You are Jarvas, a helpful AI assistant."},
	}
	for _, h := range history {
		role := openai.ChatMessageRoleUser
		if h.Role == entity.RoleAssistant {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: h.Content})
	}
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userMsg})
	return msgs
}

// ── DTO mappers ───────────────────────────────────────────────────────────────

func toConvDTO(c *entity.Conversation) *dto.ConversationResponse {
	r := &dto.ConversationResponse{
		ID:        c.ID.String(),
		Title:     c.Title,
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
	if c.AgentID != nil {
		r.AgentID = c.AgentID.String()
	}
	return r
}

func toMsgDTO(m *entity.Message) *dto.MessageResponse {
	return &dto.MessageResponse{
		ID:        m.ID.String(),
		Role:      string(m.Role),
		Content:   m.Content,
		Model:     m.Model,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
}
