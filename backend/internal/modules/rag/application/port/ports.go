package port

import (
	"context"

	"github.com/google/uuid"
)

// EmbeddingPort generates vector embeddings from text.
type EmbeddingPort interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// VectorPoint is the data stored in the vector database.
type VectorPoint struct {
	ID      uuid.UUID
	Vector  []float32
	Payload map[string]interface{}
}

// SearchResult is one match returned from ANN search.
type SearchResult struct {
	ID      uuid.UUID
	Score   float32
	Payload map[string]interface{}
}

// VectorStorePort abstracts the vector database (Qdrant in production).
type VectorStorePort interface {
	EnsureCollection(ctx context.Context, name string, dim uint64) error
	Upsert(ctx context.Context, collection string, points []VectorPoint) error
	Search(ctx context.Context, collection string, query []float32, filter map[string]string, topK uint64, minScore float32) ([]SearchResult, error)
	DeleteByFilter(ctx context.Context, collection string, filter map[string]string) error
}
