package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/agent/application/dto"
	"github.com/jarvas/backend/internal/modules/agent/application/port"
	"github.com/jarvas/backend/internal/modules/agent/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type AgentService struct {
	repo port.AgentRepository
}

func NewAgentService(repo port.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

func (s *AgentService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateAgentRequest) (*dto.AgentResponse, error) {
	agentType := entity.AgentType(req.Type)
	a := entity.NewAgent(userID, req.Name, agentType)
	a.Description = req.Description
	if req.SystemPrompt != "" {
		a.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		a.Model = req.Model
	}
	if req.Temperature != 0 {
		a.Temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		a.MaxTokens = req.MaxTokens
	}
	a.ToolsEnabled = req.ToolsEnabled
	a.MemoryEnabled = req.MemoryEnabled
	a.RAGEnabled = req.RAGEnabled

	if err := s.repo.Save(ctx, a); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toDTO(a), nil
}

func (s *AgentService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.AgentResponse, int64, error) {
	agents, total, err := s.repo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	var result []*dto.AgentResponse
	for _, a := range agents {
		result = append(result, toDTO(a))
	}
	return result, total, nil
}

func (s *AgentService) GetByID(ctx context.Context, agentID, userID uuid.UUID) (*dto.AgentResponse, error) {
	a, err := s.repo.FindByID(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}
	return toDTO(a), nil
}

func (s *AgentService) Update(ctx context.Context, agentID, userID uuid.UUID, req dto.UpdateAgentRequest) (*dto.AgentResponse, error) {
	a, err := s.repo.FindByID(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		a.Name = req.Name
	}
	if req.Description != "" {
		a.Description = req.Description
	}
	if req.SystemPrompt != "" {
		a.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		a.Model = req.Model
	}
	if req.Temperature != 0 {
		a.Temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		a.MaxTokens = req.MaxTokens
	}
	if req.ToolsEnabled != nil {
		a.ToolsEnabled = req.ToolsEnabled
	}
	a.MemoryEnabled = req.MemoryEnabled
	a.RAGEnabled = req.RAGEnabled
	a.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toDTO(a), nil
}

func (s *AgentService) Delete(ctx context.Context, agentID, userID uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, agentID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, agentID)
}

// FindAgentByID is used by the runner service — no user ownership check.
func (s *AgentService) FindAgentByID(ctx context.Context, agentID uuid.UUID) (*entity.Agent, error) {
	// Reuse repo.FindByID but with zero UUID for userID to bypass ownership.
	// We implement a direct lookup via the port.
	return s.repo.FindByID(ctx, agentID, uuid.Nil)
}

func toDTO(a *entity.Agent) *dto.AgentResponse {
	tools := a.ToolsEnabled
	if tools == nil {
		tools = []string{}
	}
	return &dto.AgentResponse{
		ID:            a.ID.String(),
		Name:          a.Name,
		Description:   a.Description,
		Type:          string(a.Type),
		Model:         a.Model,
		Temperature:   a.Temperature,
		MaxTokens:     a.MaxTokens,
		ToolsEnabled:  tools,
		MemoryEnabled: a.MemoryEnabled,
		RAGEnabled:    a.RAGEnabled,
		IsActive:      a.IsActive,
		CreatedAt:     a.CreatedAt.Format(time.RFC3339),
	}
}
