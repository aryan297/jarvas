package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/memory/application/dto"
	"github.com/jarvas/backend/internal/modules/memory/application/port"
	"github.com/jarvas/backend/internal/modules/memory/domain/entity"
	ragport "github.com/jarvas/backend/internal/modules/rag/application/port"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

const CollectionMemory = "memory"

// embeddingDim must match the model used (text-embedding-3-small → 1536).
const embeddingDim = 1536

type MemoryService struct {
	repo        port.MemoryRepository
	embedder    ragport.EmbeddingPort
	vectorStore ragport.VectorStorePort
	oai         *openai.Client
	model       string
}

func NewMemoryService(
	repo port.MemoryRepository,
	embedder ragport.EmbeddingPort,
	vectorStore ragport.VectorStorePort,
	openAIKey, model string,
) *MemoryService {
	return &MemoryService{
		repo:        repo,
		embedder:    embedder,
		vectorStore: vectorStore,
		oai:         openai.NewClient(openAIKey),
		model:       model,
	}
}

// EnsureCollection creates the Qdrant memory collection if it does not exist.
func (s *MemoryService) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, CollectionMemory, embeddingDim)
}

// Create saves a new memory entry and upserts its vector into Qdrant.
func (s *MemoryService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateMemoryRequest) (*dto.MemoryResponse, error) {
	importance := req.Importance
	if importance == 0 {
		importance = 0.5
	}

	m := entity.NewMemory(userID, entity.MemoryType(req.Type), req.Content, importance)
	if err := s.repo.Save(ctx, m); err != nil {
		return nil, apperrors.Internal(err)
	}

	go s.indexVector(context.Background(), m)

	return toDTO(m), nil
}

// List returns paginated memories for a user.
func (s *MemoryService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.MemoryResponse, int64, error) {
	memories, total, err := s.repo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	var result []*dto.MemoryResponse
	for _, m := range memories {
		result = append(result, toDTO(m))
	}
	return result, total, nil
}

// Delete removes a memory from Postgres and Qdrant.
func (s *MemoryService) Delete(ctx context.Context, memoryID, userID uuid.UUID) error {
	m, err := s.repo.FindByID(ctx, memoryID, userID)
	if err != nil {
		return err
	}
	if m.QdrantID != nil {
		_ = s.vectorStore.DeleteByFilter(ctx, CollectionMemory, map[string]string{
			"memory_id": m.ID.String(),
		})
	}
	return s.repo.Delete(ctx, memoryID)
}

// Search runs semantic search over a user's memories.
func (s *MemoryService) Search(ctx context.Context, userID uuid.UUID, req dto.SearchMemoryRequest) ([]dto.MemorySearchResult, error) {
	topK := req.TopK
	if topK == 0 {
		topK = 5
	}
	minScore := req.MinScore
	if minScore == 0 {
		minScore = 0.3
	}

	vec, err := s.embedder.EmbedText(ctx, req.Query)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("embed query: %w", err))
	}

	results, err := s.vectorStore.Search(ctx, CollectionMemory, vec,
		map[string]string{"user_id": userID.String()},
		uint64(topK), minScore)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("vector search: %w", err))
	}

	var out []dto.MemorySearchResult
	for _, r := range results {
		memID, _ := uuid.Parse(str(r.Payload["memory_id"]))
		_ = s.repo.RecordAccess(ctx, memID)
		out = append(out, dto.MemorySearchResult{
			ID:         memID.String(),
			Type:       str(r.Payload["type"]),
			Content:    str(r.Payload["content"]),
			Importance: float64(r.Score),
			Score:      r.Score,
		})
	}
	return out, nil
}

// SearchRelevant returns the content strings of the top-k most relevant memories.
// Implements the chat.port.MemoryRetriever interface.
func (s *MemoryService) SearchRelevant(ctx context.Context, userID uuid.UUID, query string, limit int) ([]string, error) {
	results, err := s.Search(ctx, userID, dto.SearchMemoryRequest{
		Query: query, TopK: limit, MinScore: 0.4,
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.Content)
	}
	return out, nil
}

// Extract uses the LLM to pull 0–3 memory facts from a conversation turn,
// then saves them. Called asynchronously after each chat reply.
func (s *MemoryService) Extract(ctx context.Context, userID uuid.UUID, userMsg, assistMsg string) error {
	prompt := buildExtractionPrompt(userMsg, assistMsg)

	resp, err := s.oai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     s.model,
		MaxTokens: 400,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: extractionSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return fmt.Errorf("extraction llm call: %w", err)
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Strip markdown code fences if present.
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var extracted []extractedFact
	if err := json.Unmarshal([]byte(raw), &extracted); err != nil {
		return nil // LLM returned nothing useful — not an error
	}

	for _, f := range extracted {
		if f.Content == "" {
			continue
		}
		importance := f.Importance
		if importance < 0.1 {
			importance = 0.5
		}
		m := entity.NewMemory(userID, entity.MemoryType(f.Type), f.Content, importance)
		if err := s.repo.Save(ctx, m); err != nil {
			continue
		}
		go s.indexVector(context.Background(), m)
	}
	return nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

func (s *MemoryService) indexVector(ctx context.Context, m *entity.Memory) {
	vec, err := s.embedder.EmbedText(ctx, m.Content)
	if err != nil {
		return
	}

	pointID := uuid.New()
	err = s.vectorStore.Upsert(ctx, CollectionMemory, []ragport.VectorPoint{{
		ID:     pointID,
		Vector: vec,
		Payload: map[string]interface{}{
			"memory_id":  m.ID.String(),
			"user_id":    m.UserID.String(),
			"type":       string(m.Type),
			"content":    m.Content,
			"importance": m.Importance,
			"created_at": m.CreatedAt.Format(time.RFC3339),
		},
	}})
	if err != nil {
		return
	}

	_ = s.repo.UpdateQdrantID(ctx, m.ID, pointID)
}

type extractedFact struct {
	Type       string  `json:"type"`
	Content    string  `json:"content"`
	Importance float64 `json:"importance"`
}

const extractionSystemPrompt = `You are a memory extraction assistant. Given a single conversation exchange, extract 0–3 durable facts about the USER (not the assistant). Only extract facts that are clearly stated and worth remembering long-term.

Return a JSON array. Each element:
{"type":"FACT|PREFERENCE|SKILL|EVENT|RELATIONSHIP","content":"...","importance":0.0-1.0}

Rules:
- If nothing is worth remembering, return []
- type must be one of: FACT, PREFERENCE, SKILL, EVENT, RELATIONSHIP
- importance: 0.9=critical personal detail, 0.7=strong preference, 0.5=useful fact, 0.3=minor detail
- content should be phrased as a fact about the user, e.g. "User prefers Python over Java"
- Do NOT extract things the assistant said, only facts about the user`

func buildExtractionPrompt(userMsg, assistMsg string) string {
	return fmt.Sprintf("User: %s\n\nAssistant: %s", userMsg, assistMsg)
}

func toDTO(m *entity.Memory) *dto.MemoryResponse {
	return &dto.MemoryResponse{
		ID:          m.ID.String(),
		Type:        string(m.Type),
		Content:     m.Content,
		Importance:  m.Importance,
		AccessCount: m.AccessCount,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
	}
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
