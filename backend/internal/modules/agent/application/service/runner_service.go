package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatenity "github.com/jarvas/backend/internal/modules/chat/domain/entity"
	einorunner "github.com/jarvas/backend/internal/modules/agent/infrastructure/eino"
	"github.com/jarvas/backend/internal/modules/agent/application/port"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// MemoryRetriever is a minimal interface matching memory.MemoryService.SearchRelevant.
type MemoryRetriever interface {
	SearchRelevant(ctx context.Context, userID uuid.UUID, query string, limit int) ([]string, error)
}

// RunnerService executes agent turns using the Eino runner.
// It implements the chat.port.AgentRunnerPort interface.
type RunnerService struct {
	agentRepo  port.AgentRepository
	registry   *toolsvc.Registry
	einoRunner *einorunner.Runner
	openAIKey  string
	memSvc     MemoryRetriever // optional
}

func NewRunnerService(
	agentRepo port.AgentRepository,
	registry *toolsvc.Registry,
	openAIKey string,
) *RunnerService {
	return &RunnerService{
		agentRepo:  agentRepo,
		registry:   registry,
		einoRunner: einorunner.NewRunner(),
		openAIKey:  openAIKey,
	}
}

func (s *RunnerService) SetMemoryRetriever(m MemoryRetriever) {
	s.memSvc = m
}

// RunMessage is the non-streaming path.
func (s *RunnerService) RunMessage(ctx context.Context, agentID, userID uuid.UUID, userMsg string, history []chatenity.Message) (string, error) {
	cfg, err := s.buildRunConfig(ctx, agentID, userID, userMsg, history)
	if err != nil {
		return "", err
	}
	result, err := s.einoRunner.Run(ctx, *cfg)
	if err != nil {
		return "", apperrors.Internal(fmt.Errorf("agent run: %w", err))
	}
	return result, nil
}

// StreamMessage is the streaming path — returns a channel of delta strings.
func (s *RunnerService) StreamMessage(ctx context.Context, agentID, userID uuid.UUID, userMsg string, history []chatenity.Message) (<-chan string, error) {
	cfg, err := s.buildRunConfig(ctx, agentID, userID, userMsg, history)
	if err != nil {
		return nil, err
	}
	ch, err := s.einoRunner.Stream(ctx, *cfg)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("agent stream: %w", err))
	}
	return ch, nil
}

// ── Private ────────────────────────────────────────────────────────────────

func (s *RunnerService) buildRunConfig(ctx context.Context, agentID, userID uuid.UUID, userMsg string, history []chatenity.Message) (*einorunner.RunConfig, error) {
	// Use uuid.Nil as userID in FindByID so any user can invoke their agent via chat.
	agent, err := s.agentRepo.FindByID(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}

	systemPrompt := buildAgentPrompt(agent.SystemPrompt, agent.Name)

	// Inject long-term memory if the agent has it enabled.
	if agent.MemoryEnabled && s.memSvc != nil {
		memories, _ := s.memSvc.SearchRelevant(ctx, userID, userMsg, 5)
		if len(memories) > 0 {
			var sb strings.Builder
			sb.WriteString(systemPrompt)
			sb.WriteString("\n\n## What you know about this user:\n")
			for _, m := range memories {
				sb.WriteString("- ")
				sb.WriteString(m)
				sb.WriteString("\n")
			}
			systemPrompt = sb.String()
		}
	}

	toolInfos := s.registry.InfoFor(agent.ToolsEnabled)

	return &einorunner.RunConfig{
		OpenAIKey:    s.openAIKey,
		Model:        agent.Model,
		Temperature:  agent.Temperature,
		MaxTokens:    agent.MaxTokens,
		SystemPrompt: systemPrompt,
		History:      history,
		UserMsg:      userMsg,
		Tools:        toolInfos,
		Registry:     s.registry,
	}, nil
}

func buildAgentPrompt(custom, name string) string {
	if custom != "" {
		return custom
	}
	return fmt.Sprintf("You are %s, a helpful AI assistant.", name)
}
