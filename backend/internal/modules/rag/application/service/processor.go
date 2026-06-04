package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	docentity "github.com/jarvas/backend/internal/modules/document/domain/entity"
	docevent "github.com/jarvas/backend/internal/modules/document/domain/event"
	docport "github.com/jarvas/backend/internal/modules/document/application/port"
	"github.com/jarvas/backend/internal/modules/document/infrastructure/extractor"
	ragport "github.com/jarvas/backend/internal/modules/rag/application/port"
	"github.com/jarvas/backend/internal/shared/eventbus"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

const (
	CollectionDocuments = "documents"
	EmbeddingDimensions = 1536 // text-embedding-3-small
	ChunkSize           = 512  // tokens per chunk
	ChunkOverlap        = 50   // overlap tokens
)

// Processor runs the full RAG indexing pipeline for a document.
// It is triggered by the DocumentUploaded event and runs in a goroutine.
type Processor struct {
	docRepo     docport.DocumentRepository
	chunkRepo   docport.ChunkRepository
	storage     docport.StoragePort
	embedder    ragport.EmbeddingPort
	vectorStore ragport.VectorStorePort
	bus         *eventbus.Bus
}

func NewProcessor(
	docRepo docport.DocumentRepository,
	chunkRepo docport.ChunkRepository,
	storage docport.StoragePort,
	embedder ragport.EmbeddingPort,
	vectorStore ragport.VectorStorePort,
	bus *eventbus.Bus,
) *Processor {
	return &Processor{
		docRepo:     docRepo,
		chunkRepo:   chunkRepo,
		storage:     storage,
		embedder:    embedder,
		vectorStore: vectorStore,
		bus:         bus,
	}
}

// EnsureCollection creates the Qdrant collection on startup if needed.
func (p *Processor) EnsureCollection(ctx context.Context) error {
	return p.vectorStore.EnsureCollection(ctx, CollectionDocuments, EmbeddingDimensions)
}

// ProcessDocument is the full pipeline: download → extract → chunk → embed → upsert.
func (p *Processor) ProcessDocument(ctx context.Context, docID, userID uuid.UUID) {
	log := logger.With(zap.String("doc_id", docID.String()))

	// 1. Mark PROCESSING
	if err := p.docRepo.UpdateStatus(ctx, docID, docentity.StatusProcessing, ""); err != nil {
		log.Error("processor: update status failed", zap.Error(err))
		return
	}

	fail := func(msg string, err error) {
		log.Error("processor: "+msg, zap.Error(err))
		_ = p.docRepo.UpdateStatus(ctx, docID, docentity.StatusFailed, msg+": "+err.Error())
	}

	// 2. Load metadata
	doc, err := p.docRepo.FindByID(ctx, docID, userID)
	if err != nil {
		fail("load document", err)
		return
	}

	// 3. Download raw file from MinIO
	reader, err := p.storage.Download(ctx, doc.StorageKey)
	if err != nil {
		fail("download from storage", err)
		return
	}
	defer reader.Close()

	// 4. Read into buffer (needed for pdf.NewReader which requires ReaderAt)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		fail("read file", err)
		return
	}

	// 5. Extract text
	ext := extractor.ForMIME(doc.MimeType)
	text, err := ext.Extract(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		fail("extract text", err)
		return
	}
	if text == "" {
		fail("extract text", fmt.Errorf("no text content found"))
		return
	}

	// 6. Chunk
	chunks := ChunkText(text, ChunkSize, ChunkOverlap)
	log.Info("processor: chunked", zap.Int("chunks", len(chunks)))

	// 7. Embed in batches of 100 (OpenAI limit: 2048)
	const batchSize = 100
	var dbChunks []*docentity.DocumentChunk
	var vectorPoints []ragport.VectorPoint

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]

		texts := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = c.Content
		}

		vecs, err := p.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			fail("embed batch", err)
			return
		}

		for j, c := range batch {
			chunkID := uuid.New()
			dbChunks = append(dbChunks, &docentity.DocumentChunk{
				ID:         chunkID,
				DocumentID: docID,
				UserID:     userID,
				Content:    c.Content,
				ChunkIndex: c.ChunkIndex,
				TokenCount: c.TokenCount,
				CreatedAt:  time.Now().UTC(),
			})
			vectorPoints = append(vectorPoints, ragport.VectorPoint{
				ID:     chunkID,
				Vector: vecs[j],
				Payload: map[string]interface{}{
					"user_id":     userID.String(),
					"document_id": docID.String(),
					"chunk_index": c.ChunkIndex,
					"content":     c.Content,
					"doc_name":    doc.Name,
				},
			})
		}
	}

	// 8. Save chunks to Postgres
	if err := p.chunkRepo.SaveBatch(ctx, dbChunks); err != nil {
		fail("save chunks", err)
		return
	}

	// 9. Upsert vectors to Qdrant
	if err := p.vectorStore.Upsert(ctx, CollectionDocuments, vectorPoints); err != nil {
		fail("upsert vectors", err)
		return
	}

	// 10. Mark INDEXED
	if err := p.docRepo.UpdateIndexed(ctx, docID, len(dbChunks)); err != nil {
		log.Error("processor: update indexed failed", zap.Error(err))
	}

	log.Info("processor: document indexed", zap.Int("chunks", len(dbChunks)))

	// 11. Publish event
	p.bus.Publish(ctx, docevent.DocumentIndexed{
		DocumentID: docID,
		ChunkCount: len(dbChunks),
		OccuredAt:  time.Now().UTC(),
	})
}
