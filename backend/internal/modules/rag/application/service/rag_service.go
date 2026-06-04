package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	ragport "github.com/jarvas/backend/internal/modules/rag/application/port"
	"github.com/jarvas/backend/internal/modules/rag/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type RAGService struct {
	embedder    ragport.EmbeddingPort
	vectorStore ragport.VectorStorePort
}

func NewRAGService(embedder ragport.EmbeddingPort, vectorStore ragport.VectorStorePort) *RAGService {
	return &RAGService{embedder: embedder, vectorStore: vectorStore}
}

// Search runs the full retrieval pipeline: embed → ANN search → rerank → build context.
func (s *RAGService) Search(ctx context.Context, query entity.SearchQuery) (*entity.RAGContext, error) {
	// 1. Embed the query
	queryVec, err := s.embedder.EmbedText(ctx, query.Query)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("embed query: %w", err))
	}

	// 2. Build filter: always scope to user_id for tenant isolation
	filter := map[string]string{"user_id": query.UserID.String()}

	topK := query.TopK
	if topK == 0 {
		topK = 20
	}
	minScore := query.MinScore
	if minScore == 0 {
		minScore = 0.3
	}

	// 3. ANN search in Qdrant
	results, err := s.vectorStore.Search(ctx, CollectionDocuments, queryVec, filter, uint64(topK), minScore)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("vector search: %w", err))
	}
	if len(results) == 0 {
		return &entity.RAGContext{Chunks: nil, TotalTokens: 0}, nil
	}

	// 4. Map to RetrievedChunk (enrich from payload — no extra DB call needed)
	chunks := make([]entity.RetrievedChunk, 0, len(results))
	for _, r := range results {
		docID, _ := uuid.Parse(str(r.Payload["document_id"]))
		chunks = append(chunks, entity.RetrievedChunk{
			ChunkID:     r.ID,
			DocumentID:  docID,
			Content:     str(r.Payload["content"]),
			DocName:     str(r.Payload["doc_name"]),
			Score:       r.Score,
			RerankScore: r.Score, // reranking placeholder — cross-encoder goes here
		})
	}

	// 5. Rerank: sort by score descending, keep top-5
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].RerankScore > chunks[j].RerankScore
	})
	limit := 5
	if len(chunks) < limit {
		limit = len(chunks)
	}
	chunks = chunks[:limit]

	// 6. Build context with token budget (≈3000 tokens = 12000 chars)
	return buildContext(chunks, 12000), nil
}

// BuildContextString formats the RAGContext into a system-prompt-ready string.
func BuildContextString(rc *entity.RAGContext) string {
	if rc == nil || len(rc.Chunks) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Relevant knowledge from your documents:\n\n")
	for i, c := range rc.Chunks {
		sb.WriteString(fmt.Sprintf("**[%d] %s**\n%s\n\n", i+1, c.DocName, c.Content))
	}
	return sb.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildContext(chunks []entity.RetrievedChunk, maxChars int) *entity.RAGContext {
	var used []entity.RetrievedChunk
	total := 0
	for _, c := range chunks {
		if total+len(c.Content) > maxChars {
			break
		}
		used = append(used, c)
		total += len(c.Content)
	}
	return &entity.RAGContext{Chunks: used, TotalTokens: total / 4}
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
