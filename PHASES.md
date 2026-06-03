# Jarvas — Phase-by-Phase Development Guide

> A step-by-step build plan for every phase after Phase 1 (Auth ✅ Done).
> Each phase follows the same pattern: Domain → Port → Service → Repository → Handler → Routes → Wire → Frontend.

---

## Status Tracker

| Phase | Feature          | Backend | Frontend | Status |
|-------|------------------|---------|----------|--------|
| 1     | Auth             | ✅ Done  | ✅ Done   | **Complete** |
| 2     | Chat             | 🔲       | 🔲        | **Next** |
| 3     | Documents + RAG  | 🔲       | 🔲        | Pending |
| 4     | Memory           | 🔲       | 🔲        | Pending |
| 5     | AI Agents (Eino) | 🔲       | 🔲        | Pending |
| 6     | Voice            | 🔲       | 🔲        | Pending |
| 7     | Workflows + Tools| 🔲       | 🔲        | Pending |
| 8     | Multi-Tenant     | 🔲       | 🔲        | Pending |

---

## How to Read This File

Each phase has:
1. **Goal** — what you're building and why
2. **Files to create** — exact paths, in the order you should create them
3. **Key code** — the critical logic to implement (not boilerplate)
4. **Wire step** — what to add to `cmd/server/main.go`
5. **Test it** — curl commands to verify before moving on
6. **Frontend steps** — components and hooks to build

---

---

# Phase 2 — Chat

**Goal:** Users can create conversations, send messages, and receive AI responses.
Basic streaming via SSE. Short-term memory in Redis.

**Time estimate:** 4–5 days

---

## Step 1 — Backend: Port (interface)

**File:** `backend/internal/modules/chat/application/port/repository.go`

```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/jarvas/backend/internal/modules/chat/domain/entity"
)

type ConversationRepository interface {
    Create(ctx context.Context, conv *entity.Conversation) error
    FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Conversation, error)
    FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Conversation, int64, error)
    UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
    Archive(ctx context.Context, id uuid.UUID) error
}

type MessageRepository interface {
    Save(ctx context.Context, msg *entity.Message) error
    FindByConversationID(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*entity.Message, error)
    CountByConversationID(ctx context.Context, convID uuid.UUID) (int64, error)
}
```

---

## Step 2 — Backend: Service

**File:** `backend/internal/modules/chat/application/service/chat_service.go`

```go
package service

type ChatConfig struct {
    OpenAIKey      string
    Model          string
    MaxTokens      int
    ShortTermLimit int   // how many messages to keep in Redis
}

type ChatService struct {
    convRepo  port.ConversationRepository
    msgRepo   port.MessageRepository
    cache     *cache.Client
    bus       *eventbus.Bus
    cfg       ChatConfig
}

// Key logic to implement:
//
// CreateConversation — insert row, return DTO
//
// SendMessage(ctx, userID, req) → MessageResponse:
//   1. Get or create conversation
//   2. Save user message to DB
//   3. Load recent messages from Redis (short-term memory)
//   4. If cache miss → load last N from DB, warm cache
//   5. Build messages slice for OpenAI
//   6. Call OpenAI chat completion (non-streaming first, add stream later)
//   7. Save assistant message to DB
//   8. Update Redis cache (append + trim to limit)
//   9. Publish ChatCompleted event
//  10. Return assistant message DTO
//
// StreamMessage — same as SendMessage but returns chan string
//   Use OpenAI streaming API, write SSE frames
```

**Redis short-term memory key pattern:**
```
short_term:{conversation_id}   →  JSON []Message   TTL: 2h sliding
```

**OpenAI call (non-streaming):**
```go
import "github.com/sashabaranov/go-openai"

client := openai.NewClient(cfg.OpenAIKey)
resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
    Model:     cfg.Model,
    Messages:  buildMessages(systemPrompt, history, userMsg),
    MaxTokens: cfg.MaxTokens,
})
content := resp.Choices[0].Message.Content
```

**Add to go.mod:**
```bash
go get github.com/sashabaranov/go-openai
```

---

## Step 3 — Backend: Repositories

**File:** `backend/internal/modules/chat/infrastructure/repository/conversation_repo.go`

```go
// Key queries:
//
// Create: INSERT INTO conversations (id, user_id, agent_id, title, status, created_at, updated_at)
//
// FindByUserID: SELECT ... FROM conversations WHERE user_id=$1 AND status='ACTIVE'
//              ORDER BY updated_at DESC LIMIT $2 OFFSET $3
//
// Archive: UPDATE conversations SET status='ARCHIVED' WHERE id=$1 AND user_id=$2
```

**File:** `backend/internal/modules/chat/infrastructure/repository/message_repo.go`

```go
// Key queries:
//
// Save: INSERT INTO messages (id, conversation_id, role, content, token_count, model, created_at)
//
// FindByConversationID: SELECT ... FROM messages WHERE conversation_id=$1
//                       ORDER BY created_at ASC LIMIT $2 OFFSET $3
//
// IMPORTANT: Scan role column as string → entity.MessageRole (same pattern as user role in auth)
```

---

## Step 4 — Backend: Handler + Routes

**File:** `backend/internal/modules/chat/delivery/http/handler.go`

```go
// Endpoints to implement:
//
// CreateConversation  POST   /conversations
// ListConversations   GET    /conversations       ?page=1&limit=20
// GetConversation     GET    /conversations/:id
// DeleteConversation  DELETE /conversations/:id
// SendMessage         POST   /conversations/:id/messages   body: {content, stream}
// ListMessages        GET    /conversations/:id/messages   ?page=1&limit=50

// Streaming handler skeleton:
func (h *ChatHandler) SendMessage(c *gin.Context) {
    // ... parse request, get userID from context

    if req.Stream {
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("Connection", "keep-alive")
        c.Header("X-Accel-Buffering", "no")   // disable nginx buffering

        flusher := c.Writer.(http.Flusher)
        ch := h.chatSvc.StreamMessage(c.Request.Context(), userID, req)
        for chunk := range ch {
            fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
            flusher.Flush()
        }
        fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
        flusher.Flush()
        return
    }

    // non-streaming path
    msg, err := h.chatSvc.SendMessage(c.Request.Context(), userID, req)
    if err != nil { response.Error(c, err); return }
    response.Created(c, msg)
}
```

---

## Step 5 — Wire in main.go

```go
// Add after auth wiring:
import (
    chatrepo    "github.com/jarvas/backend/internal/modules/chat/infrastructure/repository"
    chatsvc     "github.com/jarvas/backend/internal/modules/chat/application/service"
    chathttp    "github.com/jarvas/backend/internal/modules/chat/delivery/http"
)

convRepo := chatrepo.NewConversationRepository(db.Pool)
msgRepo  := chatrepo.NewMessageRepository(db.Pool)
chatSvc  := chatsvc.NewChatService(convRepo, msgRepo, redisClient, bus, chatsvc.ChatConfig{
    OpenAIKey: cfg.AI.OpenAIKey,
    Model:     cfg.AI.Model,
    MaxTokens: 4096,
    ShortTermLimit: 20,
})
chatHandler := chathttp.NewChatHandler(chatSvc)
chathttp.RegisterRoutes(v1, chatHandler, authMW)
```

---

## Step 6 — Test It

```bash
# Create conversation
curl -X POST :8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "My first chat"}'

# Send message
curl -X POST :8080/api/v1/conversations/$CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello! What can you do?"}'

# Stream response
curl -N -X POST :8080/api/v1/conversations/$CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Explain Go interfaces", "stream": true}'
```

---

## Step 7 — Frontend: Chat

**Files to create:**

```
frontend/src/
├── services/chat.service.ts         # API calls
├── hooks/useChat.ts                 # React Query + streaming hook
├── pages/chat/
│   ├── ChatPage.tsx                 # Layout: ConversationList + MessageThread
│   ├── ConversationList.tsx         # Left sidebar: list of conversations
│   ├── MessageThread.tsx            # Right panel: messages + input box
│   ├── MessageBubble.tsx            # Single message with markdown rendering
│   └── ChatInput.tsx                # Textarea + send button + stream toggle
```

**chat.service.ts key methods:**
```typescript
createConversation: (data) => apiClient.post('/conversations', data)
listConversations:  ()     => apiClient.get('/conversations')
sendMessage:        (convId, data) => apiClient.post(`/conversations/${convId}/messages`, data)

// For streaming — use native EventSource or fetch with ReadableStream:
streamMessage: (convId, content, onChunk: (text: string) => void, onDone: () => void) => {
  fetch(`/api/v1/conversations/${convId}/messages`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ content, stream: true }),
  }).then(res => {
    const reader = res.body!.getReader()
    const decoder = new TextDecoder()
    const pump = () => reader.read().then(({ done, value }) => {
      if (done) { onDone(); return }
      const text = decoder.decode(value)
      text.split('\n').filter(l => l.startsWith('data:')).forEach(line => {
        const d = line.slice(5).trim()
        if (d !== '[DONE]') onChunk(d)
      })
      pump()
    })
    pump()
  })
}
```

---

---

# Phase 3 — Documents + RAG

**Goal:** Users upload documents → system chunks + embeds them → AI can search and cite them.

**Time estimate:** 5–7 days

**New dependencies to add:**
```bash
go get github.com/minio/minio-go/v7
go get github.com/qdrant/go-client
go get google.golang.org/grpc
```

---

## Step 1 — MinIO Storage Client

**File:** `backend/internal/modules/document/infrastructure/storage/minio_storage.go`

```go
package storage

import (
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
    client *minio.Client
    bucket string
}

func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil { return nil, err }

    // Ensure bucket exists
    ctx := context.Background()
    exists, _ := client.BucketExists(ctx, bucket)
    if !exists {
        client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
    }
    return &MinIOStorage{client: client, bucket: bucket}, nil
}

// Key methods to implement:
// Upload(ctx, key, reader, size, contentType) error
// GetPresignedURL(ctx, key, expiry) (string, error)
// Delete(ctx, key) error
// Download(ctx, key) (io.ReadCloser, error)
```

---

## Step 2 — Qdrant Vector Store

**File:** `backend/internal/modules/rag/infrastructure/vectorstore/qdrant.go`

```go
package vectorstore

import (
    pb     "github.com/qdrant/go-client/qdrant"
    "google.golang.org/grpc"
)

type QdrantStore struct {
    client     pb.PointsClient
    collection string
}

func NewQdrantStore(host string, port int, collection string) (*QdrantStore, error) {
    conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithInsecure())
    // ...

    // Create collection if not exists (1536 dims, Cosine)
    // pb.CollectionsClient.Create(...)
}

// Key methods:
//
// UpsertPoints(ctx, points []VectorPoint) error
//   VectorPoint = { ID uuid, Vector []float32, Payload map[string]any }
//
// Search(ctx, query []float32, userID uuid, topK int, minScore float64) ([]SearchResult, error)
//   Filter: must match payload.user_id = userID (tenant isolation)
//
// DeleteByDocumentID(ctx, documentID uuid) error
//   Filter delete: payload.document_id = documentID
```

---

## Step 3 — Embedding Service

**File:** `backend/internal/modules/rag/infrastructure/embedding/openai_embedding.go`

```go
package embedding

// EmbedText calls OpenAI text-embedding-3-small, returns []float32 (1536 dims)
func (e *OpenAIEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
    resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.AdaEmbeddingV2,  // or text-embedding-3-small
    })
    if err != nil { return nil, err }
    return resp.Data[0].Embedding, nil
}

// EmbedBatch — embed multiple texts in one API call (rate limit aware)
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
```

---

## Step 4 — Document Chunker

**File:** `backend/internal/modules/rag/application/service/chunker.go`

```go
package service

// Chunking strategy: fixed-size with overlap
// ChunkSize = 512 tokens, Overlap = 50 tokens
//
// Algorithm:
// 1. Tokenise text (rough estimate: 1 token ≈ 4 chars)
// 2. Split into sentences (split on ". ", "? ", "! ", "\n\n")
// 3. Accumulate sentences until ChunkSize reached
// 4. Start next chunk with last Overlap tokens from previous chunk
// 5. Each chunk gets chunk_index for ordering

type Chunk struct {
    Content    string
    ChunkIndex int
    TokenCount int
}

func ChunkText(text string, chunkSize, overlap int) []Chunk
```

---

## Step 5 — Text Extractors

**File:** `backend/internal/modules/document/infrastructure/extractor/extractor.go`

```go
// Interface:
type Extractor interface {
    Extract(reader io.Reader) (string, error)
}

// Implementations to build:
// PDFExtractor   — use "github.com/ledongthuc/pdf"
// TXTExtractor   — just io.ReadAll
// MDExtractor    — just io.ReadAll (treat as plain text)
// DOCXExtractor  — use "github.com/carmel/gooxml"

// Factory:
func NewExtractor(mimeType string) Extractor {
    switch mimeType {
    case "application/pdf":  return &PDFExtractor{}
    case "text/plain":       return &TXTExtractor{}
    case "text/markdown":    return &MDExtractor{}
    default:                 return &TXTExtractor{}
    }
}
```

---

## Step 6 — RAG Pipeline (Processor)

**File:** `backend/internal/modules/rag/application/service/processor.go`

```go
// Called as a goroutine when DocumentUploaded event fires.
// Full pipeline:

func (p *Processor) ProcessDocument(ctx context.Context, docID uuid.UUID) error {
    // 1. Load document metadata from DB
    doc, _ := p.docRepo.FindByID(ctx, docID)

    // 2. Mark status = PROCESSING
    p.docRepo.UpdateStatus(ctx, docID, entity.StatusProcessing)

    // 3. Download raw file from MinIO
    reader, _ := p.storage.Download(ctx, doc.StorageKey)
    defer reader.Close()

    // 4. Extract text
    extractor := NewExtractor(doc.MimeType)
    text, _ := extractor.Extract(reader)

    // 5. Chunk the text
    chunks := ChunkText(text, 512, 50)

    // 6. Embed + upsert each chunk
    for i, chunk := range chunks {
        vector, _ := p.embedder.EmbedText(ctx, chunk.Content)

        // Save chunk row in Postgres
        dbChunk := entity.DocumentChunk{
            ID:         uuid.New(),
            DocumentID: docID,
            UserID:     doc.UserID,
            Content:    chunk.Content,
            ChunkIndex: i,
            TokenCount: chunk.TokenCount,
        }
        p.chunkRepo.Save(ctx, &dbChunk)

        // Upsert vector in Qdrant
        p.vectorStore.UpsertPoints(ctx, []VectorPoint{{
            ID:      dbChunk.ID,
            Vector:  vector,
            Payload: map[string]any{
                "user_id":     doc.UserID.String(),
                "document_id": docID.String(),
                "chunk_index": i,
                "content":     chunk.Content,  // stored for hybrid retrieval
            },
        }})
    }

    // 7. Mark status = INDEXED
    p.docRepo.UpdateStatusIndexed(ctx, docID, len(chunks))

    // 8. Publish EmbeddingCreated event
    p.bus.Publish(ctx, event.EmbeddingCreated{DocumentID: docID})

    return nil
}
```

---

## Step 7 — RAG Search (Retrieval)

**File:** `backend/internal/modules/rag/application/service/rag_service.go`

```go
func (s *RAGService) Search(ctx context.Context, query entity.SearchQuery) (*entity.RAGContext, error) {
    // 1. Embed the query
    queryVec, _ := s.embedder.EmbedText(ctx, query.Query)

    // 2. Qdrant ANN search (filtered by user_id)
    results, _ := s.vectorStore.Search(ctx, queryVec, query.UserID, query.TopK, query.MinScore)

    // 3. Load chunk metadata from Postgres (doc name, page number etc.)
    chunks := enrichFromDB(ctx, results)

    // 4. Rerank (simple: sort by score desc; advanced: cross-encoder)
    ranked := rerank(chunks, 5)

    // 5. Build context string with token budget
    return buildContext(ranked, 3000), nil  // max 3000 tokens of context
}

// buildContext assembles the retrieved chunks into a formatted string:
// ---
// [Source: document_name.pdf, chunk 3]
// <chunk content>
// ---
// This gets injected into the system prompt before the user's question.
```

---

## Step 8 — Document Handler

**File:** `backend/internal/modules/document/delivery/http/handler.go`

```go
// Upload — multipart form
func (h *DocumentHandler) Upload(c *gin.Context) {
    file, header, _ := c.Request.FormFile("file")
    defer file.Close()

    // Validate size (max 50MB) and mime type
    if header.Size > 50<<20 { /* reject */ }

    // Generate storage key
    storageKey := fmt.Sprintf("users/%s/%s/%s", userID, uuid.New(), header.Filename)

    // Upload to MinIO
    h.storage.Upload(ctx, storageKey, file, header.Size, header.Header.Get("Content-Type"))

    // Create DB record
    doc := entity.NewDocument(userID, header.Filename, mimeType, storageKey, header.Size)
    h.docRepo.Create(ctx, doc)

    // Publish event → triggers async processing
    h.bus.Publish(ctx, event.DocumentUploaded{DocumentID: doc.ID, UserID: userID})

    response.Created(c, toDTO(doc))
}
```

---

## Step 9 — Subscribe to Events in main.go

```go
// Register event handlers in main.go (after wiring services):
processor := ragservice.NewProcessor(docRepo, chunkRepo, storage, embedder, qdrant, bus)

bus.Subscribe(documentevent.EvtDocumentUploaded, func(ctx context.Context, e eventbus.Event) {
    evt := e.(documentevent.DocumentUploaded)
    go processor.ProcessDocument(ctx, evt.DocumentID)
})
```

---

## Step 10 — Test It

```bash
# Upload a document
curl -X POST :8080/api/v1/documents \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/document.pdf"

# Check processing status
curl :8080/api/v1/documents/$DOC_ID \
  -H "Authorization: Bearer $TOKEN"
# → {"status": "INDEXED", "chunk_count": 42}

# RAG search
curl -X POST :8080/api/v1/rag/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "what is the refund policy?", "top_k": 5}'
```

---

## Step 11 — Frontend: Documents

```
frontend/src/
├── services/document.service.ts
├── pages/documents/
│   ├── DocumentsPage.tsx       # grid of uploaded documents with status badges
│   ├── DocumentCard.tsx        # card: name, type, status, chunk count, delete
│   ├── UploadDropzone.tsx      # react-dropzone: drag & drop or click to upload
│   └── DocumentSearch.tsx      # search box → RAG search → display results
```

**Key UX notes:**
- Show a processing spinner on cards with status `PROCESSING`
- Poll `GET /documents/:id` every 3s while status is `UPLOADED` or `PROCESSING`
- RAG search results show source document name + snippet

---

---

# Phase 4 — Memory

**Goal:** AI remembers facts about the user across conversations. Short-term (Redis), long-term (Postgres + Qdrant).

**Time estimate:** 3–4 days

---

## Step 1 — Memory Service

**File:** `backend/internal/modules/memory/application/service/memory_service.go`

```go
// CRUD operations (user can view/edit/delete their memories)
func (s *MemoryService) Create(ctx, userID, req) (*entity.Memory, error)
func (s *MemoryService) List(ctx, userID, filter) ([]*entity.Memory, error)
func (s *MemoryService) Update(ctx, id, userID, req) error
func (s *MemoryService) Delete(ctx, id, userID) error

// Semantic search (used by agent before responding)
func (s *MemoryService) Search(ctx, userID, query string, topK int) ([]*entity.Memory, error) {
    // 1. Embed query
    // 2. Qdrant search filtered by user_id on "memory" collection
    // 3. RecordAccess() on each returned memory (increment counter, update timestamp)
    // 4. Return sorted by rerankScore
}
```

---

## Step 2 — Auto-Extractor

**File:** `backend/internal/modules/memory/application/service/extractor.go`

```go
// Triggered by ChatCompleted event.
// Uses LLM to extract structured memories from the last conversation turn.

func (e *Extractor) ExtractFromTurn(ctx context.Context, userMsg, assistantMsg string, userID uuid.UUID) error {
    prompt := `You are a memory extraction assistant. Given this conversation turn, extract important
    personal information worth remembering long-term. Return JSON array:
    [{"type":"FACT|PREFERENCE|EVENT|SKILL|RELATIONSHIP","content":"...","importance":0.0-1.0}]
    Only extract clearly stated facts, not assumptions. Return [] if nothing worth saving.`

    result, _ := llm.Complete(ctx, prompt, userMsg+"\n"+assistantMsg)
    memories := parseJSON(result)

    for _, m := range memories {
        mem := entity.NewMemory(userID, m.Type, m.Content, m.Importance)
        s.memRepo.Save(ctx, mem)
        vector, _ := s.embedder.EmbedText(ctx, m.Content)
        s.vectorStore.UpsertPoints(ctx, ...)
        s.bus.Publish(ctx, event.MemoryCreated{MemoryID: mem.ID})
    }
    return nil
}

// Subscribe in main.go:
bus.Subscribe(chatevent.EvtChatCompleted, func(ctx context.Context, e eventbus.Event) {
    evt := e.(chatevent.ChatCompleted)
    go extractor.ExtractFromTurn(ctx, evt.UserMessage, evt.AssistantMessage, evt.UserID)
})
```

---

## Step 3 — Qdrant: Second Collection

Create the `memory` collection separately from `documents`:

```go
// In qdrant.go or a setup script:
// Collection: "memory"  — same 1536 dims, Cosine distance
// Payload fields: user_id (keyword), type (keyword), importance (float)
```

---

## Step 4 — Wire Memory into Chat

In `chat_service.go → SendMessage`, before calling OpenAI:

```go
// Load relevant memories
memories, _ := s.memSvc.Search(ctx, userID, userMessage, 5)

// Inject into system prompt
memoryContext := formatMemories(memories)
systemPrompt = basePrompt + "\n\n## What you know about this user:\n" + memoryContext
```

---

## Step 5 — Test It

```bash
# List memories
curl :8080/api/v1/memories \
  -H "Authorization: Bearer $TOKEN"

# Create memory manually
curl -X POST :8080/api/v1/memories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"PREFERENCE","content":"User prefers Python over Java","importance":0.8}'

# Semantic memory search
curl -X POST :8080/api/v1/memories/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"programming preferences","top_k":3}'
```

---

---

# Phase 5 — AI Agents (Eino)

**Goal:** Users create named agents with custom system prompts. Supervisor routes to sub-agents. Full tool calling.

**Time estimate:** 7–10 days

**New dependency:**
```bash
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

---

## Step 1 — Add go.mod dependencies

```go
require (
    github.com/cloudwego/eino     v0.3.21
    github.com/cloudwego/eino-ext v0.3.21
)
```

---

## Step 2 — Agent Repository

**File:** `backend/internal/modules/agent/infrastructure/repository/agent_repo.go`

```go
// Key queries:
// Create:     INSERT INTO agents (id, user_id, name, type, system_prompt, model, ...)
// FindByID:   SELECT ... FROM agents WHERE id=$1 AND user_id=$2
// FindByUser: SELECT ... FROM agents WHERE user_id=$1 AND is_active=TRUE ORDER BY created_at DESC
// Update:     UPDATE agents SET name=$2, system_prompt=$3, ... WHERE id=$1 AND user_id=$2
// Delete:     UPDATE agents SET is_active=FALSE WHERE id=$1 AND user_id=$2
```

---

## Step 3 — Eino Graph: Simple Agent First

**File:** `backend/internal/modules/agent/infrastructure/eino/simple_agent.go`

```go
package eino

import (
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// SimpleAgent wraps a single LLM node with tool support.
// Build this first before the Supervisor pattern.

func BuildSimpleAgent(cfg AgentConfig) (compose.Runnable, error) {
    graph := compose.NewGraph[*AgentInput, *AgentOutput]()

    // LLM node
    model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:       cfg.Model,
        APIKey:      cfg.OpenAIKey,
        Temperature: &cfg.Temperature,
        MaxTokens:   &cfg.MaxTokens,
    })

    graph.AddChatModelNode("llm", model)
    graph.AddEdge(compose.START, "llm")
    graph.AddEdge("llm", compose.END)

    return graph.Compile()
}

// Run with context injection:
func (a *SimpleAgent) Run(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
    messages := buildMessages(input.SystemPrompt, input.History, input.UserMessage)
    return a.runnable.Invoke(ctx, &AgentInput{Messages: messages})
}
```

---

## Step 4 — Supervisor Graph

**File:** `backend/internal/modules/agent/infrastructure/eino/supervisor_graph.go`

```go
// Build this after SimpleAgent works end-to-end.
//
// Pattern: Supervisor decides which sub-agent handles the request.
//
// Graph topology:
//   START → supervisor_llm → router_lambda
//   router_lambda → research_agent  (if type="research")
//   router_lambda → coding_agent    (if type="coding")
//   router_lambda → planning_agent  (if type="planning")
//   research_agent  → supervisor_llm  (for multi-turn)
//   coding_agent    → supervisor_llm
//   planning_agent  → supervisor_llm
//   supervisor_llm  → END            (when done)

graph.AddConditionalEdges("router", routeByType, map[string]string{
    "research":  "research_agent",
    "coding":    "coding_agent",
    "planning":  "planning_agent",
    compose.END: compose.END,
})
```

---

## Step 5 — Tool Registration

**File:** `backend/internal/modules/tool/application/service/tool_registry.go`

```go
// The registry maps tool names → Eino tool nodes.
// Adding a new tool = implement interface + register here.

type Registry struct {
    executors map[string]entity.Executor
}

func (r *Registry) Register(name string, exec entity.Executor) {
    r.executors[name] = exec
}

// Built-in tools to implement first:

// HTTPTool — make arbitrary HTTP GET/POST requests
// PostgresTool — run read-only SELECT queries against user's configured DB
// GoogleCalendarTool — list/create events (Phase 7)
// GitHubTool — list repos, read files, create issues (Phase 7)
// EmailTool — send via SMTP (Phase 7)

// Convert to Eino tool node:
func (r *Registry) ToEinoTool(name string) eino.Tool {
    exec := r.executors[name]
    return eino.NewTool(exec.Schema(), func(ctx, input) (output, error) {
        return exec.Execute(ctx, input)
    })
}
```

---

## Step 6 — Agent Runner Service

**File:** `backend/internal/modules/agent/application/service/agent_runner.go`

```go
func (r *AgentRunner) Run(ctx context.Context, agentID, userID uuid.UUID, convID uuid.UUID, userMessage string) (*RunResult, error) {
    // 1. Load agent config from DB
    agent, _ := r.agentRepo.FindByID(ctx, agentID, userID)

    // 2. Load conversation history (short-term memory from Redis)
    history, _ := r.chatSvc.LoadHistory(ctx, convID)

    // 3. Retrieve relevant memories (if memory_enabled)
    var memCtx string
    if agent.MemoryEnabled {
        mems, _ := r.memSvc.Search(ctx, userID, userMessage, 5)
        memCtx = formatMemories(mems)
    }

    // 4. Retrieve RAG context (if rag_enabled)
    var ragCtx string
    if agent.RAGEnabled {
        ragResult, _ := r.ragSvc.Search(ctx, entity.SearchQuery{
            UserID: userID, Query: userMessage, TopK: 5, MinScore: 0.5,
        })
        ragCtx = ragResult.String()
    }

    // 5. Build system prompt
    systemPrompt := buildSystemPrompt(agent.SystemPrompt, memCtx, ragCtx)

    // 6. Get enabled tools
    tools := r.registry.GetTools(agent.ToolsEnabled)

    // 7. Build and run Eino graph
    graph := r.buildGraph(agent, tools)
    result, _ := graph.Invoke(ctx, &AgentInput{
        SystemPrompt: systemPrompt,
        History:      history,
        UserMessage:  userMessage,
    })

    return result, nil
}
```

---

## Step 7 — Test It

```bash
# Create an agent
curl -X POST :8080/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Research Assistant",
    "type": "RESEARCH",
    "system_prompt": "You are a helpful research assistant...",
    "model": "gpt-4o",
    "memory_enabled": true,
    "rag_enabled": true,
    "tools_enabled": ["http_request"]
  }'

# Chat with the agent
curl -X POST :8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "$AGENT_ID"}'

curl -X POST :8080/api/v1/conversations/$CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "What did I upload last week?"}'
```

---

---

# Phase 6 — Voice

**Goal:** User records audio → backend transcribes via Whisper → injects as message → AI responds.

**Time estimate:** 2–3 days

---

## Step 1 — Audio Storage

Use the existing MinIO `audio` bucket.

**Storage key pattern:** `audio/{user_id}/{session_id}.webm`

---

## Step 2 — Whisper Integration

**File:** `backend/internal/modules/voice/infrastructure/transcription/whisper.go`

```go
package transcription

import (
    "github.com/sashabaranov/go-openai"
)

func (w *WhisperTranscriber) Transcribe(ctx context.Context, audioKey string) (string, error) {
    // 1. Download audio from MinIO
    reader, _ := w.storage.Download(ctx, audioKey)
    defer reader.Close()

    // 2. Call OpenAI Whisper API
    resp, err := w.client.CreateTranscription(ctx, openai.AudioRequest{
        Model:    openai.Whisper1,
        Reader:   reader,
        FilePath: "audio.webm",   // extension tells Whisper the format
    })
    if err != nil { return "", err }
    return resp.Text, nil
}
```

---

## Step 3 — Voice Service

**File:** `backend/internal/modules/voice/application/service/voice_service.go`

```go
func (s *VoiceService) UploadAndTranscribe(ctx context.Context, userID, convID uuid.UUID, file io.Reader, size int64) (*entity.VoiceSession, error) {
    // 1. Upload audio to MinIO
    key := fmt.Sprintf("audio/%s/%s.webm", userID, uuid.New())
    s.storage.Upload(ctx, key, file, size, "audio/webm")

    // 2. Create voice session row
    session := entity.NewVoiceSession(userID, convID, key)
    s.sessionRepo.Save(ctx, session)

    // 3. Transcribe async
    go func() {
        text, err := s.transcriber.Transcribe(ctx, key)
        if err != nil {
            s.sessionRepo.UpdateStatus(ctx, session.ID, entity.TranscriptionFailed)
            return
        }
        s.sessionRepo.UpdateTranscript(ctx, session.ID, text)

        // 4. Inject transcript as user message into conversation
        s.chatSvc.SendMessage(ctx, userID, dto.SendMessageRequest{
            ConversationID: convID.String(),
            Content:        text,
        })
    }()

    return session, nil
}
```

---

## Step 4 — Routes

```
POST /api/v1/voice/upload              multipart: file, conversation_id
GET  /api/v1/voice/sessions/:id        poll transcription status
GET  /api/v1/voice/sessions            list sessions
```

---

## Step 5 — Frontend: Voice Button

```
frontend/src/components/
└── VoiceRecorder.tsx    # MediaRecorder API → stop → POST /voice/upload → poll status
```

```typescript
// VoiceRecorder key logic:
const startRecording = async () => {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  const recorder = new MediaRecorder(stream, { mimeType: 'audio/webm' })
  const chunks: Blob[] = []
  recorder.ondataavailable = (e) => chunks.push(e.data)
  recorder.onstop = async () => {
    const blob = new Blob(chunks, { type: 'audio/webm' })
    const form = new FormData()
    form.append('file', blob, 'recording.webm')
    form.append('conversation_id', convId)
    const res = await apiClient.post('/voice/upload', form)
    // poll GET /voice/sessions/:id until status=COMPLETED
    pollTranscription(res.data.data.id)
  }
  recorder.start()
  setRecorder(recorder)
}
```

---

---

# Phase 7 — Workflows + Tools

**Goal:** Users build automated multi-step workflows triggered manually, by schedule, or by events.

**Time estimate:** 7–10 days

---

## Step 1 — Workflow Engine (DAG Executor)

**File:** `backend/internal/modules/workflow/infrastructure/engine/dag_executor.go`

```go
// WorkflowDefinition has nodes + edges. Execute via topological sort.

func (e *DAGExecutor) Execute(ctx context.Context, workflow *entity.Workflow, payload map[string]any) error {
    // 1. Create WorkflowRun row (status=RUNNING)
    run := entity.WorkflowRun{...}
    e.runRepo.Save(ctx, &run)

    // 2. Topological sort of nodes (walk edges from START)
    order := topologicalSort(workflow.Definition)

    // 3. Execute each node in order
    vars := map[string]any{"input": payload}
    for _, nodeID := range order {
        node := findNode(workflow.Definition.Nodes, nodeID)
        result, err := e.executeNode(ctx, node, vars)
        if err != nil {
            e.runRepo.UpdateStatus(ctx, run.ID, entity.RunFailed, err.Error())
            return err
        }
        vars["output_"+nodeID] = result
    }

    // 4. Update run status = COMPLETED
    e.runRepo.UpdateStatus(ctx, run.ID, entity.RunCompleted, nil)
    e.bus.Publish(ctx, event.WorkflowExecuted{RunID: run.ID})
    return nil
}

// executeNode dispatches to the right executor based on node.Type:
func (e *DAGExecutor) executeNode(ctx context.Context, node entity.WorkflowNode, vars map[string]any) (any, error) {
    switch node.Type {
    case "agent":
        return e.agentRunner.Run(ctx, node.Config["agent_id"], ...)
    case "tool":
        return e.toolRegistry.Execute(ctx, node.Config["tool"], vars)
    case "condition":
        return e.evalCondition(node.Config, vars)
    case "delay":
        time.Sleep(parseDuration(node.Config["duration"]))
        return nil, nil
    }
    return nil, fmt.Errorf("unknown node type: %s", node.Type)
}
```

---

## Step 2 — Scheduled Workflow Trigger

**File:** `backend/internal/modules/workflow/infrastructure/scheduler/scheduler.go`

```go
// Use a simple goroutine-based cron (or "github.com/robfig/cron/v3"):
// go get github.com/robfig/cron/v3

import "github.com/robfig/cron/v3"

type Scheduler struct {
    cron    *cron.Cron
    entries map[uuid.UUID]cron.EntryID
}

func (s *Scheduler) AddWorkflow(wf *entity.Workflow) error {
    id, err := s.cron.AddFunc(wf.CronExpr, func() {
        s.engine.Execute(ctx, wf, map[string]any{"trigger": "schedule"})
    })
    s.entries[wf.ID] = id
    return err
}

func (s *Scheduler) RemoveWorkflow(wfID uuid.UUID) {
    s.cron.Remove(s.entries[wfID])
    delete(s.entries, wfID)
}
```

---

## Step 3 — Built-in Tool Executors

**File:** `backend/internal/modules/tool/infrastructure/executors/`

```
http_tool.go          — net/http GET/POST with configurable headers
postgres_tool.go      — read-only query against user-configured Postgres
github_tool.go        — GitHub REST API v3 (list repos, read file, create issue)
google_calendar_tool.go — Google Calendar API (list events, create event)
email_tool.go         — SMTP send via net/smtp
```

**Each tool follows this pattern:**

```go
type HTTPTool struct{}

func (t *HTTPTool) Name() string { return "http_request" }

func (t *HTTPTool) Execute(ctx context.Context, input map[string]any) (any, error) {
    url := input["url"].(string)
    method := input["method"].(string)
    // ... make request, return response body
}

func (t *HTTPTool) Schema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "url":    map[string]any{"type": "string"},
            "method": map[string]any{"type": "string", "enum": []string{"GET","POST","PUT","DELETE"}},
            "body":   map[string]any{"type": "string"},
        },
        "required": []string{"url", "method"},
    }
}
```

---

## Step 4 — Workflow Routes

```
POST   /api/v1/workflows              Create
GET    /api/v1/workflows              List
GET    /api/v1/workflows/:id          Get (+ definition)
PATCH  /api/v1/workflows/:id          Update
DELETE /api/v1/workflows/:id          Delete
PATCH  /api/v1/workflows/:id/status   Activate / pause
POST   /api/v1/workflows/:id/run      Manual trigger
GET    /api/v1/workflows/:id/runs     List runs
GET    /api/v1/workflows/runs/:run_id Get run detail + result

GET    /api/v1/tools                  List available tools
GET    /api/v1/tools/:id/schema       Get tool input schema
POST   /api/v1/tools/:id/config       Save user credentials for a tool
```

---

---

# Phase 8 — Multi-Tenant

**Goal:** Multiple organisations share one deployment. Each tenant's data is isolated.

**Time estimate:** 5–7 days

---

## Step 1 — Tenant Table

```sql
-- New migration: 012_create_tenants.sql
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    plan       VARCHAR(50)  NOT NULL DEFAULT 'FREE',
    owner_id   UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_members (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    role      VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, user_id)
);
```

---

## Step 2 — Tenant Middleware

**File:** `backend/internal/shared/middleware/tenant.go`

```go
func TenantMiddleware(tenantRepo port.TenantRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetHeader("X-Tenant-ID")
        userID   := c.GetString("user_id")

        if tenantID == "" {
            c.Next()   // personal account — no tenant
            return
        }

        // Verify user is a member of this tenant
        ok, _ := tenantRepo.IsMember(ctx, tenantID, userID)
        if !ok {
            response.Error(c, apperrors.Forbidden("not a member of this tenant"))
            c.Abort()
            return
        }

        c.Set("tenant_id", tenantID)
        c.Next()
    }
}
```

---

## Step 3 — Add tenant_id to Every Query

Update every repository's queries to include `tenant_id`:

```go
// Before (personal):
SELECT * FROM documents WHERE user_id = $1

// After (tenant-aware):
SELECT * FROM documents
WHERE user_id = $1
  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
```

Or use Postgres Row Level Security for strict enforcement:

```sql
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON documents USING (
    tenant_id = current_setting('app.current_tenant', TRUE)::uuid
    OR current_setting('app.current_tenant', TRUE) = ''
);
```

---

## Step 4 — Per-Tenant Rate Limiting

```go
// Redis key: rate:{tenant_id}:{user_id}
// Instead of: rate:{ip}

func rateLimitKey(c *gin.Context) string {
    tenantID := c.GetString("tenant_id")
    userID   := c.GetString("user_id")
    if tenantID != "" {
        return fmt.Sprintf("rate:%s:%s", tenantID, userID)
    }
    return fmt.Sprintf("rate:personal:%s", userID)
}
```

---

---

# Implementation Rules (for every phase)

Follow these every time you add a new module. The auth module is the reference implementation.

## The 8-Step Pattern

```
1. domain/entity/     — Aggregate root, no external imports
2. domain/event/      — Domain events this module publishes
3. application/port/  — Repository + service interfaces (contracts)
4. application/dto/   — Request/response structs with validate tags
5. application/service/ — Business logic, depends only on ports
6. infrastructure/repository/ — pgx implementation
7. delivery/http/     — Gin handler + routes
8. cmd/server/main.go — Wire everything together
```

## Repository Rules

- Nullable DB columns → scan with `*string` / `*time.Time`, not plain types
- Enum columns → cast in SQL: `$1::my_enum`, scan as `string`, cast to typed const after scan
- Empty strings that map to nullable columns → convert to `nil` before insert

## Error Handling Rules

```go
// In repository — return raw error, let service decide
if err != nil { return nil, err }

// In service — wrap in AppError
if err != nil { return nil, apperrors.Internal(err) }

// Distinguish "not found" from real errors:
if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 { /* expected */ }

// In handler — just pass to response.Error
response.Error(c, err)  // unwraps AppError automatically, logs 5xx
```

## Go.mod: Add deps only when the phase needs them

```bash
# Phase 2
go get github.com/sashabaranov/go-openai

# Phase 3
go get github.com/minio/minio-go/v7
go get github.com/qdrant/go-client
go get google.golang.org/grpc

# Phase 5
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext

# Phase 7
go get github.com/robfig/cron/v3
```

---

*Last updated: Phase 1 complete. Start Phase 2.*
