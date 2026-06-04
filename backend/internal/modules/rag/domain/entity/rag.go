package entity

import "github.com/google/uuid"

// RetrievedChunk is the result of a vector similarity search + reranking.
type RetrievedChunk struct {
	ChunkID     uuid.UUID
	DocumentID  uuid.UUID
	Content     string
	DocName     string  // document file name for citations
	Score       float32 // cosine similarity from Qdrant
	RerankScore float32 // cross-encoder score; equals Score until Phase 5 cross-encoder
}

// RAGContext is the assembled prompt context ready for injection into the LLM.
type RAGContext struct {
	Chunks      []RetrievedChunk
	TotalTokens int
}

// SearchQuery is the input to the RAG retrieval pipeline.
type SearchQuery struct {
	UserID      uuid.UUID
	Query       string
	TopK        int
	MinScore    float32
	DocumentIDs []uuid.UUID // optional — filter to specific documents
}
