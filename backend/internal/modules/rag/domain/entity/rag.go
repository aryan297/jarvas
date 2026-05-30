package entity

import "github.com/google/uuid"

// RetrievedChunk is the result of a vector similarity search, combined with
// a reranking score. The RAG pipeline uses this to build context windows.
type RetrievedChunk struct {
	ChunkID    uuid.UUID
	DocumentID uuid.UUID
	Content    string
	Score      float64 // cosine similarity from Qdrant
	RerankScore float64 // cross-encoder score after reranking
}

// RAGContext is the assembled prompt context ready for the LLM.
type RAGContext struct {
	Chunks     []RetrievedChunk
	TotalTokens int
}

// SearchQuery is the input to the RAG retrieval pipeline.
type SearchQuery struct {
	UserID    uuid.UUID
	Query     string
	TopK      int
	MinScore  float64
	DocumentIDs []uuid.UUID // optional filter to specific documents
}
