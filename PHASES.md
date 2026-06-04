# Jarvas — Phase-by-Phase Development Guide

> Step-by-step build plan. Follow the 8-step pattern for every module.
> Reference implementation: `backend/internal/modules/auth/`

---

## Status Tracker

| Phase | Feature           | Backend      | Frontend     | Status           |
|-------|-------------------|--------------|--------------|------------------|
| 1     | Auth              | ✅ Complete   | ✅ Complete   | **DONE**         |
| 2     | Chat + Streaming  | ✅ Complete   | ✅ Complete   | **DONE**         |
| 3     | Documents + RAG   | ✅ Complete   | ✅ Complete   | **DONE**         |
| 4     | Memory            | 🔲 Not started | 🔲 Not started | **▶ ACTIVE NEXT**|
| 5     | AI Agents (Eino)  | 🔲 Not started | 🔲 Not started | Pending          |
| 6     | Voice             | 🔲 Not started | 🔲 Not started | Pending          |
| 7     | Workflows + Tools | 🔲 Not started | 🔲 Not started | Pending          |
| 8     | Multi-Tenant      | 🔲 Not started | 🔲 Not started | Pending          |

---

## The 8-Step Module Pattern

Use this order every time. Never skip steps.

```
Step 1 → domain/entity/       Aggregate root, zero external imports
Step 2 → domain/event/        Domain events this module publishes
Step 3 → application/port/    Repository interfaces (contracts)
Step 4 → application/dto/     Request/response structs + validate tags
Step 5 → application/service/ Business logic, depends only on ports
Step 6 → infrastructure/      pgx repos, Redis, MinIO, Qdrant clients
Step 7 → delivery/http/       Gin handler + route mounting
Step 8 → cmd/server/main.go   Wire everything (constructor injection)
```

---

## Critical Rules (learned from Phase 1 bugs)

### 1. Postgres Enums — always cast explicitly

```go
// BAD — pgx v5 won't auto-cast string to custom enum
VALUES ($1, $2, $3, $4)  // where $4 maps to a user_role column

// GOOD — explicit cast in SQL
VALUES ($1, $2, $3, $4::user_role)
// and pass string(entity.UserRole) not entity.UserRole
```

### 2. Nullable columns — never scan into plain types

```go
// BAD — panics if column value is NULL
var ip string
row.Scan(..., &ip, ...)

// GOOD — use pointer, then dereference after scan
var ip *string
row.Scan(..., &ip, ...)
if ip != nil { entity.IP = *ip }
```

### 3. INET / custom PG types — cast to text in SELECT

```go
// BAD — pgx v5 cannot scan INET directly into *string
SELECT ip_address FROM refresh_tokens WHERE ...

// GOOD — cast in query
SELECT ip_address::text FROM refresh_tokens WHERE ...
```

### 4. Empty string → NULL for nullable VARCHAR

```go
// Helper used in every repo that has nullable string columns
func nullStr(s string) interface{} {
    if s == "" { return nil }
    return s
}
// Use: nullStr(entity.ProviderID)  in INSERT args
```

### 5. Service error handling pattern

```go
// Repository returns raw error
// Service wraps in AppError:
if err != nil {
    if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
        // expected "not found" — handle it
    }
    return nil, apperrors.Internal(err)
}

// Handler just passes to response.Error — it unwraps automatically
response.Error(c, err)
```

---

---

# ✅ Phase 1 — Auth (COMPLETE)

**Completed:** All 11 test scenarios passing. No internal errors.

## What Was Built

| File | Status |
|------|--------|
| `auth/domain/entity/user.go` | ✅ |
| `auth/domain/entity/refresh_token.go` | ✅ |
| `auth/domain/event/events.go` | ✅ |
| `auth/application/port/repository.go` | ✅ |
| `auth/application/dto/auth_dto.go` | ✅ |
| `auth/application/service/auth_service.go` | ✅ |
| `auth/application/service/token_service.go` | ✅ |
| `auth/infrastructure/repository/user_repo.go` | ✅ |
| `auth/infrastructure/repository/token_repo.go` | ✅ |
| `auth/infrastructure/oauth/google.go` | ✅ |
| `auth/delivery/http/handler.go` | ✅ |
| `auth/delivery/http/routes.go` | ✅ |
| `shared/middleware/auth.go` | ✅ |
| `frontend/src/pages/auth/LoginPage.tsx` (tabbed: login + register) | ✅ |

## Verified Endpoints

| # | Endpoint | Expected | Result |
|---|----------|----------|--------|
| 1 | `POST /auth/register` | 201 + tokens + user | ✅ |
| 2 | `POST /auth/login` | 200 + tokens + user | ✅ |
| 3 | `GET /auth/me` | 200 + full profile | ✅ |
| 4 | `POST /auth/refresh` | 200 + new pair, old revoked | ✅ |
| 5 | Old refresh token reuse | 401 revoked | ✅ |
| 6 | `/me` with rotated token | 200 | ✅ |
| 7 | `POST /auth/logout` | 204 | ✅ |
| 8 | Duplicate email | 409 CONFLICT | ✅ |
| 9 | Wrong password | 401 UNAUTHORIZED | ✅ |
| 10 | Invalid JWT | 401 UNAUTHORIZED | ✅ |
| 11 | Short password (< 8 chars) | 400 BAD_REQUEST | ✅ |

## Bugs Fixed During Phase 1

| Bug | Cause | Fix Applied |
|-----|-------|-------------|
| Register → 500 | pgx v5 can't cast string → Postgres enum | Added `$6::user_role` cast in INSERT |
| Register → 500 | `INET` column rejects empty string `""` | `nullStr()` converts `""` to `nil`; handler passes real IP |
| Refresh → "not found" | pgx v5 can't scan `INET` into `*string` | Added `ip_address::text` cast in SELECT |
| `/me` empty full_name | JWT claims didn't carry `full_name` | Added `name` field to `Claims`; middleware sets `user_name` |
| Provider ID unique violation | Empty string `""` ≠ NULL for partial index | `nullStr()` on `provider_id` INSERT |

---

---

# ✅ Phase 2 — Chat (COMPLETE)

**Built:** Conversations, messages, OpenAI chat completion, SSE streaming, short-term memory in Redis.

## Files Built

```
Backend
  ✅ chat/application/port/repository.go           ConversationRepository, MessageRepository interfaces
  ✅ chat/application/dto/chat_dto.go              Request/response DTOs
  ✅ chat/application/service/chat_service.go      SendMessage, StreamMessage, history via Redis
  ✅ chat/infrastructure/repository/conversation_repo.go
  ✅ chat/infrastructure/repository/message_repo.go
  ✅ chat/delivery/http/handler.go                 6 endpoints + SSE streaming handler
  ✅ chat/delivery/http/routes.go

Frontend
  ✅ services/chat.service.ts                      API + native fetch SSE streaming
  ✅ hooks/useConversations.ts                     React Query hooks
  ✅ pages/chat/ChatPage.tsx                       Split layout: list + thread
  ✅ pages/chat/ConversationList.tsx               New/delete, active highlight, timestamps
  ✅ pages/chat/MessageThread.tsx                  Optimistic UI, streaming accumulator, auto-scroll
  ✅ pages/chat/MessageBubble.tsx                  Markdown rendering, streaming cursor ▍
  ✅ pages/chat/ChatInput.tsx                      Auto-resize, stream toggle, Enter to send
```

## Verified Endpoints

```
POST   /api/v1/conversations              Create conversation
GET    /api/v1/conversations              List (paginated)
GET    /api/v1/conversations/:id          Get + messages
DELETE /api/v1/conversations/:id          Archive
POST   /api/v1/conversations/:id/messages Send (non-stream + SSE stream)
GET    /api/v1/conversations/:id/messages List messages (paginated)
```

## Key Design Decisions

- **Auto-create conversation**: `SendMessage` creates one on the fly if no `conversation_id` given
- **Short-term memory**: Redis key `short_term:{conv_id}` holds last 20 messages, TTL 2h sliding
- **Streaming**: native `fetch` + `ReadableStream` (not Axios) → server-sent events `data: delta\n\n`
- **Activate**: set `OPENAI_API_KEY` in `.env`

---

## Step 1 — Port

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

## Step 2 — DTO (extend existing)

**File:** `backend/internal/modules/chat/application/dto/chat_dto.go` (already exists — add these)

```go
// Add to existing file:

type ListConversationsRequest struct {
    Page  int `form:"page"  binding:"min=1"`
    Limit int `form:"limit" binding:"min=1,max=100"`
}

type ListMessagesRequest struct {
    Page  int `form:"page"  binding:"min=1"`
    Limit int `form:"limit" binding:"min=1,max=200"`
}
```

---

## Step 3 — Service

**File:** `backend/internal/modules/chat/application/service/chat_service.go`

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    openai "github.com/sashabaranov/go-openai"
    "github.com/google/uuid"
    "github.com/jarvas/backend/internal/modules/chat/application/dto"
    "github.com/jarvas/backend/internal/modules/chat/application/port"
    "github.com/jarvas/backend/internal/modules/chat/domain/entity"
    "github.com/jarvas/backend/internal/shared/cache"
    "github.com/jarvas/backend/internal/shared/eventbus"
    apperrors "github.com/jarvas/backend/internal/shared/errors"
)

const shortTermTTL  = 2 * time.Hour
const shortTermLimit = 20   // messages to keep in Redis

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

func NewChatService(convRepo port.ConversationRepository, msgRepo port.MessageRepository,
    cache *cache.Client, bus *eventbus.Bus, cfg ChatConfig) *ChatService {
    return &ChatService{
        convRepo: convRepo,
        msgRepo:  msgRepo,
        cache:    cache,
        bus:      bus,
        oai:      openai.NewClient(cfg.OpenAIKey),
        cfg:      cfg,
    }
}

func (s *ChatService) CreateConversation(ctx context.Context, userID uuid.UUID, req dto.CreateConversationRequest) (*dto.ConversationResponse, error) {
    var agentID *uuid.UUID
    if req.AgentID != "" {
        id, _ := uuid.Parse(req.AgentID)
        agentID = &id
    }
    conv := entity.NewConversation(userID, agentID)
    conv.Title = req.Title
    if err := s.convRepo.Create(ctx, conv); err != nil {
        return nil, apperrors.Internal(err)
    }
    return toConvDTO(conv), nil
}

func (s *ChatService) SendMessage(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (*dto.MessageResponse, error) {
    // 1. Get or create conversation
    convID, _ := uuid.Parse(req.ConversationID)
    conv, err := s.convRepo.FindByID(ctx, convID, userID)
    if err != nil {
        return nil, err
    }

    // 2. Save user message to DB
    userMsg := &entity.Message{
        ID:             uuid.New(),
        ConversationID: conv.ID,
        Role:           entity.RoleUser,
        Content:        req.Content,
        CreatedAt:      time.Now().UTC(),
    }
    s.msgRepo.Save(ctx, userMsg)

    // 3. Load short-term memory from Redis
    history := s.loadHistory(ctx, conv.ID)

    // 4. Build OpenAI messages
    oaiMessages := buildOAIMessages("You are Jarvas, a helpful AI assistant.", history, req.Content)

    // 5. Call OpenAI
    resp, err := s.oai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:     s.cfg.Model,
        Messages:  oaiMessages,
        MaxTokens: s.cfg.MaxTokens,
    })
    if err != nil {
        return nil, apperrors.Internal(fmt.Errorf("openai: %w", err))
    }

    content := resp.Choices[0].Message.Content

    // 6. Save assistant message to DB
    asstMsg := &entity.Message{
        ID:             uuid.New(),
        ConversationID: conv.ID,
        Role:           entity.RoleAssistant,
        Content:        content,
        TokenCount:     resp.Usage.TotalTokens,
        Model:          s.cfg.Model,
        CreatedAt:      time.Now().UTC(),
    }
    s.msgRepo.Save(ctx, asstMsg)

    // 7. Update Redis short-term memory
    s.appendHistory(ctx, conv.ID, userMsg, asstMsg)

    return toMsgDTO(asstMsg), nil
}

// StreamMessage — returns a channel of string delta chunks.
// The handler reads from this channel and writes SSE frames.
func (s *ChatService) StreamMessage(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (<-chan string, error) {
    convID, _ := uuid.Parse(req.ConversationID)
    conv, err := s.convRepo.FindByID(ctx, convID, userID)
    if err != nil { return nil, err }

    userMsg := &entity.Message{
        ID: uuid.New(), ConversationID: conv.ID,
        Role: entity.RoleUser, Content: req.Content, CreatedAt: time.Now().UTC(),
    }
    s.msgRepo.Save(ctx, userMsg)

    history := s.loadHistory(ctx, conv.ID)
    oaiMessages := buildOAIMessages("You are Jarvas, a helpful AI assistant.", history, req.Content)

    stream, err := s.oai.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
        Model:     s.cfg.Model,
        Messages:  oaiMessages,
        MaxTokens: s.cfg.MaxTokens,
        Stream:    true,
    })
    if err != nil { return nil, apperrors.Internal(err) }

    ch := make(chan string, 64)
    go func() {
        defer close(ch)
        defer stream.Close()

        var fullContent string
        for {
            resp, err := stream.Recv()
            if err != nil { break }
            delta := resp.Choices[0].Delta.Content
            if delta == "" { continue }
            fullContent += delta
            ch <- delta
        }

        // Save complete assistant message to DB
        asstMsg := &entity.Message{
            ID: uuid.New(), ConversationID: conv.ID,
            Role: entity.RoleAssistant, Content: fullContent,
            Model: s.cfg.Model, CreatedAt: time.Now().UTC(),
        }
        s.msgRepo.Save(ctx, asstMsg)
        s.appendHistory(ctx, conv.ID, userMsg, asstMsg)
    }()

    return ch, nil
}

// ── Short-term memory helpers ─────────────────────────────────────────────────

func (s *ChatService) loadHistory(ctx context.Context, convID uuid.UUID) []entity.Message {
    key := fmt.Sprintf("short_term:%s", convID)
    var msgs []entity.Message
    if err := s.cache.Get(ctx, key, &msgs); err == nil {
        return msgs
    }
    // Cache miss: load from DB and warm cache
    dbMsgs, _ := s.msgRepo.FindByConversationID(ctx, convID, shortTermLimit, 0)
    for _, m := range dbMsgs {
        msgs = append(msgs, *m)
    }
    s.cache.Set(ctx, key, msgs, shortTermTTL)
    return msgs
}

func (s *ChatService) appendHistory(ctx context.Context, convID uuid.UUID, msgs ...*entity.Message) {
    key := fmt.Sprintf("short_term:%s", convID)
    var history []entity.Message
    s.cache.Get(ctx, key, &history)
    for _, m := range msgs {
        history = append(history, *m)
    }
    if len(history) > shortTermLimit {
        history = history[len(history)-shortTermLimit:]
    }
    s.cache.Set(ctx, key, history, shortTermTTL)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildOAIMessages(system string, history []entity.Message, userMsg string) []openai.ChatCompletionMessage {
    msgs := []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: system}}
    for _, h := range history {
        role := openai.ChatMessageRoleUser
        if h.Role == entity.RoleAssistant { role = openai.ChatMessageRoleAssistant }
        msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: h.Content})
    }
    msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userMsg})
    return msgs
}

func toConvDTO(c *entity.Conversation) *dto.ConversationResponse {
    return &dto.ConversationResponse{
        ID:        c.ID.String(),
        Title:     c.Title,
        Status:    string(c.Status),
        CreatedAt: c.CreatedAt.Format(time.RFC3339),
        UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
    }
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

// silence unused import
var _ = json.Marshal
```

---

## Step 4 — Conversation Repository

**File:** `backend/internal/modules/chat/infrastructure/repository/conversation_repo.go`

```go
package repository

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jarvas/backend/internal/modules/chat/domain/entity"
    apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgConversationRepository struct{ db *pgxpool.Pool }

func NewConversationRepository(db *pgxpool.Pool) *pgConversationRepository {
    return &pgConversationRepository{db: db}
}

func (r *pgConversationRepository) Create(ctx context.Context, c *entity.Conversation) error {
    q := `INSERT INTO conversations (id, user_id, agent_id, title, status, created_at, updated_at)
          VALUES ($1,$2,$3,$4,$5::conversation_status,$6,$7)`
    _, err := r.db.Exec(ctx, q,
        c.ID, c.UserID, c.AgentID, nullStr(c.Title),
        string(c.Status), c.CreatedAt, c.UpdatedAt)
    return err
}

func (r *pgConversationRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Conversation, error) {
    q := `SELECT id, user_id, agent_id, title, status::text, created_at, updated_at
          FROM conversations WHERE id=$1 AND user_id=$2 AND status!='DELETED'`
    row := r.db.QueryRow(ctx, q, id, userID)
    return scanConversation(row)
}

func (r *pgConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Conversation, int64, error) {
    // Count
    var total int64
    r.db.QueryRow(ctx, `SELECT COUNT(*) FROM conversations WHERE user_id=$1 AND status='ACTIVE'`, userID).Scan(&total)

    // Data
    rows, err := r.db.Query(ctx,
        `SELECT id, user_id, agent_id, title, status::text, created_at, updated_at
         FROM conversations WHERE user_id=$1 AND status='ACTIVE'
         ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
        userID, limit, offset)
    if err != nil { return nil, 0, err }
    defer rows.Close()

    var convs []*entity.Conversation
    for rows.Next() {
        c, _ := scanConversation(rows)
        convs = append(convs, c)
    }
    return convs, total, nil
}

func (r *pgConversationRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
    _, err := r.db.Exec(ctx, `UPDATE conversations SET title=$2, updated_at=NOW() WHERE id=$1`, id, title)
    return err
}

func (r *pgConversationRepository) Archive(ctx context.Context, id uuid.UUID) error {
    _, err := r.db.Exec(ctx, `UPDATE conversations SET status='ARCHIVED', updated_at=NOW() WHERE id=$1`, id)
    return err
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type scanner interface {
    Scan(dest ...any) error
}

func scanConversation(row scanner) (*entity.Conversation, error) {
    var c entity.Conversation
    var agentID *uuid.UUID
    var title *string
    var status string
    err := row.Scan(&c.ID, &c.UserID, &agentID, &title, &status, &c.CreatedAt, &c.UpdatedAt)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) { return nil, apperrors.NotFound("conversation") }
        return nil, err
    }
    c.AgentID = agentID
    c.Status = entity.ConversationStatus(status)
    if title != nil { c.Title = *title }
    return &c, nil
}

func nullStr(s string) interface{} {
    if s == "" { return nil }
    return s
}
```

---

## Step 5 — Message Repository

**File:** `backend/internal/modules/chat/infrastructure/repository/message_repo.go`

```go
package repository

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jarvas/backend/internal/modules/chat/domain/entity"
)

type pgMessageRepository struct{ db *pgxpool.Pool }

func NewMessageRepository(db *pgxpool.Pool) *pgMessageRepository {
    return &pgMessageRepository{db: db}
}

func (r *pgMessageRepository) Save(ctx context.Context, m *entity.Message) error {
    q := `INSERT INTO messages (id, conversation_id, role, content, token_count, model, created_at)
          VALUES ($1,$2,$3::message_role,$4,$5,$6,$7)`
    _, err := r.db.Exec(ctx, q,
        m.ID, m.ConversationID, string(m.Role), m.Content,
        nullInt(m.TokenCount), nullStr(m.Model), m.CreatedAt)
    return err
}

func (r *pgMessageRepository) FindByConversationID(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
    rows, err := r.db.Query(ctx,
        `SELECT id, conversation_id, role::text, content, COALESCE(token_count,0), COALESCE(model,''), created_at
         FROM messages WHERE conversation_id=$1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
        convID, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var msgs []*entity.Message
    for rows.Next() {
        var m entity.Message
        var role string
        rows.Scan(&m.ID, &m.ConversationID, &role, &m.Content, &m.TokenCount, &m.Model, &m.CreatedAt)
        m.Role = entity.MessageRole(role)
        msgs = append(msgs, &m)
    }
    return msgs, nil
}

func (r *pgMessageRepository) CountByConversationID(ctx context.Context, convID uuid.UUID) (int64, error) {
    var n int64
    r.db.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE conversation_id=$1`, convID).Scan(&n)
    return n, nil
}

func nullInt(n int) interface{} {
    if n == 0 { return nil }
    return n
}
```

---

## Step 6 — Handler

**File:** `backend/internal/modules/chat/delivery/http/handler.go`

```go
package http

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jarvas/backend/internal/modules/chat/application/dto"
    "github.com/jarvas/backend/internal/modules/chat/application/service"
    apperrors "github.com/jarvas/backend/internal/shared/errors"
    "github.com/jarvas/backend/internal/shared/response"
    "github.com/jarvas/backend/internal/shared/validator"
)

type ChatHandler struct {
    chatSvc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
    return &ChatHandler{chatSvc: svc}
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
    var req dto.CreateConversationRequest
    c.ShouldBindJSON(&req)
    userID, _ := uuid.Parse(c.GetString("user_id"))

    conv, err := h.chatSvc.CreateConversation(c.Request.Context(), userID, req)
    if err != nil { response.Error(c, err); return }
    response.Created(c, conv)
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
    userID, _ := uuid.Parse(c.GetString("user_id"))
    page, limit := pageLimit(c)

    convs, total, err := h.chatSvc.ListConversations(c.Request.Context(), userID, limit, (page-1)*limit)
    if err != nil { response.Error(c, err); return }
    response.Paginated(c, convs, response.PaginationMeta{
        Page: page, Limit: limit, Total: total,
        TotalPages: int((total + int64(limit) - 1) / int64(limit)),
    })
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
    convID, _ := uuid.Parse(c.Param("id"))
    userID, _ := uuid.Parse(c.GetString("user_id"))

    conv, err := h.chatSvc.GetConversation(c.Request.Context(), convID, userID)
    if err != nil { response.Error(c, err); return }
    response.OK(c, conv)
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
    convID, _ := uuid.Parse(c.Param("id"))
    userID, _ := uuid.Parse(c.GetString("user_id"))

    if err := h.chatSvc.ArchiveConversation(c.Request.Context(), convID, userID); err != nil {
        response.Error(c, err); return
    }
    response.NoContent(c)
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
    convID := c.Param("id")
    userID, _ := uuid.Parse(c.GetString("user_id"))

    var req dto.SendMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperrors.BadRequest("invalid request body")); return
    }
    if err := validator.Validate(&req); err != nil { response.Error(c, err); return }
    req.ConversationID = convID

    // SSE streaming path
    if req.Stream {
        ch, err := h.chatSvc.StreamMessage(c.Request.Context(), userID, req)
        if err != nil { response.Error(c, err); return }

        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("Connection", "keep-alive")
        c.Header("X-Accel-Buffering", "no")

        flusher, ok := c.Writer.(http.Flusher)
        if !ok { response.Error(c, apperrors.Internal(nil)); return }

        for delta := range ch {
            fmt.Fprintf(c.Writer, "data: %s\n\n", delta)
            flusher.Flush()
        }
        fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
        flusher.Flush()
        return
    }

    // Non-streaming path
    msg, err := h.chatSvc.SendMessage(c.Request.Context(), userID, req)
    if err != nil { response.Error(c, err); return }
    response.Created(c, msg)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
    convID, _ := uuid.Parse(c.Param("id"))
    userID, _ := uuid.Parse(c.GetString("user_id"))
    page, limit := pageLimit(c)

    msgs, total, err := h.chatSvc.ListMessages(c.Request.Context(), convID, userID, limit, (page-1)*limit)
    if err != nil { response.Error(c, err); return }
    response.Paginated(c, msgs, response.PaginationMeta{
        Page: page, Limit: limit, Total: total,
        TotalPages: int((total + int64(limit) - 1) / int64(limit)),
    })
}

func pageLimit(c *gin.Context) (int, int) {
    page, limit := 1, 20
    fmt.Sscan(c.DefaultQuery("page", "1"), &page)
    fmt.Sscan(c.DefaultQuery("limit", "20"), &limit)
    if page < 1 { page = 1 }
    if limit < 1 || limit > 100 { limit = 20 }
    return page, limit
}
```

---

## Step 7 — Routes

**File:** `backend/internal/modules/chat/delivery/http/routes.go`

```go
package http

import (
    "github.com/gin-gonic/gin"
    "github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *ChatHandler, mw *middleware.AuthMiddleware) {
    g := rg.Group("/conversations", mw.Authenticate())
    {
        g.POST("",        h.CreateConversation)
        g.GET("",         h.ListConversations)
        g.GET("/:id",     h.GetConversation)
        g.DELETE("/:id",  h.DeleteConversation)
        g.POST("/:id/messages", h.SendMessage)
        g.GET("/:id/messages",  h.ListMessages)
    }
}
```

---

## Step 8 — Wire in main.go

```go
// Add these imports:
import (
    chatrepo "github.com/jarvas/backend/internal/modules/chat/infrastructure/repository"
    chatsvc  "github.com/jarvas/backend/internal/modules/chat/application/service"
    chathttp "github.com/jarvas/backend/internal/modules/chat/delivery/http"
)

// Add after auth wiring in main():
convRepo := chatrepo.NewConversationRepository(db.Pool)
msgRepo  := chatrepo.NewMessageRepository(db.Pool)
chatSvc  := chatsvc.NewChatService(convRepo, msgRepo, redisClient, bus, chatsvc.ChatConfig{
    OpenAIKey: cfg.AI.OpenAIKey,
    Model:     cfg.AI.Model,
    MaxTokens: 4096,
})
chatHandler := chathttp.NewChatHandler(chatSvc)
chathttp.RegisterRoutes(v1, chatHandler, authMW)
```

---

## Test It

```bash
# 1. Create a conversation
curl -X POST :8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "My first chat"}'

# 2. Send a message (non-streaming)
curl -X POST :8080/api/v1/conversations/$CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello! What is Go?"}'

# 3. Stream a response (SSE)
curl -N -X POST :8080/api/v1/conversations/$CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Explain goroutines", "stream": true}'

# 4. List conversations
curl :8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN"

# 5. List messages
curl ":8080/api/v1/conversations/$CONV_ID/messages?page=1&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Frontend: Chat

**Files to create:**

```
frontend/src/services/chat.service.ts
frontend/src/hooks/useConversations.ts
frontend/src/hooks/useMessages.ts
frontend/src/pages/chat/
├── ChatPage.tsx           ← layout: left ConversationList + right MessageThread
├── ConversationList.tsx   ← list with new chat button, delete, active highlight
├── MessageThread.tsx      ← scrollable message list + ChatInput at bottom
├── MessageBubble.tsx      ← user (right, blue) vs assistant (left, gray) with Markdown
└── ChatInput.tsx          ← textarea, send button, stream toggle, Shift+Enter = newline
```

**Key streaming implementation in ChatInput:**
```typescript
const streamMessage = async (convId: string, content: string, onChunk: (t: string) => void) => {
  const token = useAuthStore.getState().accessToken
  const res = await fetch(`/api/v1/conversations/${convId}/messages`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ content, stream: true }),
  })
  const reader = res.body!.getReader()
  const dec = new TextDecoder()
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    dec.decode(value).split('\n')
      .filter(l => l.startsWith('data:'))
      .forEach(l => { const d = l.slice(5).trim(); if (d !== '[DONE]') onChunk(d) })
  }
}
```

---

---

# ✅ Phase 3 — Documents + RAG (COMPLETE)

**Built:** Upload → chunk → embed → Qdrant → semantic search. Full async pipeline via event bus.

## Files Built

```
Backend
  ✅ document/domain/event/events.go                    DocumentUploaded, DocumentIndexed, DocumentDeleted
  ✅ document/application/port/ports.go                 DocumentRepository, ChunkRepository, StoragePort
  ✅ document/application/service/document_service.go   Upload, List, GetByID, Delete, PresignedURL
  ✅ document/infrastructure/storage/minio.go           Upload, Download, Delete, PresignedGetURL
  ✅ document/infrastructure/extractor/extractor.go     PDF (ledongthuc) + PlainText (TXT/MD/CSV/HTML)
  ✅ document/infrastructure/repository/document_repo.go
  ✅ document/infrastructure/repository/chunk_repo.go   Batch insert in transaction
  ✅ document/delivery/http/handler.go                  6 endpoints (upload multipart)
  ✅ document/delivery/http/routes.go

  ✅ rag/application/port/ports.go                      EmbeddingPort, VectorStorePort interfaces
  ✅ rag/application/service/chunker.go                 Sentence-aware 512-token chunks, 50-token overlap
  ✅ rag/application/service/processor.go               Full pipeline: download→extract→chunk→embed→upsert
  ✅ rag/application/service/rag_service.go             Embed query→ANN→rerank top-5→context builder
  ✅ rag/infrastructure/embedding/openai.go             text-embedding-3-small, EmbedText + EmbedBatch
  ✅ rag/infrastructure/vectorstore/qdrant.go           gRPC client: EnsureCollection, Upsert, Search, Delete
  ✅ rag/delivery/http/handler.go                       POST /rag/search
  ✅ rag/delivery/http/routes.go

Frontend
  ✅ services/document.service.ts                       Upload (multipart), list, delete, presign, RAG search
  ✅ pages/documents/DocumentsPage.tsx                  Tabbed: My Docs + Search; polls indexing every 5s
  ✅ pages/documents/UploadDropzone.tsx                 react-dropzone, 50 MB limit, accepted MIME types
  ✅ pages/documents/DocumentCard.tsx                   Status badge spinner, chunk count, download/delete
  ✅ pages/documents/DocumentSearch.tsx                 Semantic search → ranked results with % match
```

## Verified Endpoints

```
POST   /api/v1/documents              Upload (multipart/form-data, field: file)
GET    /api/v1/documents              List user's documents (paginated)
GET    /api/v1/documents/:id          Get document + status
DELETE /api/v1/documents/:id          Delete doc + chunks + fires Qdrant cleanup event
GET    /api/v1/documents/:id/url      Get 24h presigned MinIO download URL
GET    /api/v1/documents/:id/chunks   List raw text chunks
POST   /api/v1/rag/search             Semantic search (body: {query, top_k, min_score})
```

## Pipeline Flow

```
POST /documents (multipart)
  → validate size ≤ 50 MB
  → upload raw file to MinIO: documents/{user_id}/{uuid}/{filename}
  → INSERT documents row (status=UPLOADED)
  → bus.Publish(DocumentUploaded)         ← returns 201 here

Background goroutine (event handler):
  → UpdateStatus(PROCESSING)
  → Download from MinIO
  → Extract text (PDF or plain)
  → ChunkText(512 tokens, 50 overlap)
  → EmbedBatch(100 chunks/call, text-embedding-3-small)
  → SaveBatch chunks to Postgres (transaction)
  → Upsert vectors to Qdrant (payload: user_id, document_id, content, doc_name)
  → UpdateIndexed(chunkCount)
  → bus.Publish(DocumentIndexed)

DELETE /documents/:id
  → bus.Publish(DocumentDeleted)
  → goroutine: qdrant.DeleteByFilter({document_id: id})
```

## Key Design Decisions

- **Tenant isolation in Qdrant**: every search filters `payload.user_id` — users never see each other's chunks
- **Chunk content in Qdrant payload**: avoids extra DB round-trip on search (payload has the text)
- **Batch embedding**: 100 chunks per OpenAI call to stay within token limits
- **Activate**: `OPENAI_API_KEY`, `MINIO_ENDPOINT`, `QDRANT_HOST` in `.env`

---

---

# ▶ Phase 4 — Memory (ACTIVE NEXT)

**Goal:** AI remembers facts across conversations. Short-term (Redis), long-term (Postgres + Qdrant).

**Time estimate:** 3–4 days

## Checklist
```
Backend
  [ ] memory/application/port/repository.go
  [ ] memory/application/service/memory_service.go  (CRUD + semantic search)
  [ ] memory/application/service/extractor.go  (LLM extraction from ChatCompleted event)
  [ ] memory/infrastructure/repository/memory_repo.go
  [ ] Create Qdrant "memory" collection (separate from "documents")
  [ ] memory/delivery/http/handler.go
  [ ] memory/delivery/http/routes.go
  [ ] Subscribe ChatCompleted → extractor in main.go
  [ ] Inject memory context into chat service SendMessage

Frontend
  [ ] services/memory.service.ts
  [ ] pages/memory/MemoryPage.tsx  (list, create, delete, search)
  [ ] pages/memory/MemoryCard.tsx  (type badge, importance bar, content)
```

**Key pattern — inject into chat:**
```go
// In ChatService.SendMessage, before calling OpenAI:
memories, _ := s.memSvc.Search(ctx, userID, req.Content, 5)
if len(memories) > 0 {
    systemPrompt += "\n\n## What you know about this user:\n" + formatMemories(memories)
}
```

---

---

# Phase 5 — AI Agents (Eino)

**Goal:** Named agents with custom prompts, tool calling, supervisor routing to sub-agents.

**Time estimate:** 7–10 days

**Dependencies:**
```bash
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

## Checklist
```
Backend
  [ ] agent/application/port/repository.go
  [ ] agent/application/service/agent_service.go  (CRUD)
  [ ] agent/application/service/agent_runner.go   (builds Eino graph, injects context)
  [ ] agent/infrastructure/repository/agent_repo.go
  [ ] agent/infrastructure/eino/simple_agent.go   (build this first)
  [ ] agent/infrastructure/eino/supervisor_graph.go
  [ ] tool/application/service/tool_registry.go
  [ ] tool/infrastructure/executors/http_tool.go
  [ ] tool/infrastructure/executors/postgres_tool.go
  [ ] agent/delivery/http/handler.go
  [ ] agent/delivery/http/routes.go
  [ ] Wire all in main.go

Frontend
  [ ] services/agent.service.ts
  [ ] pages/agents/AgentsPage.tsx     (agent list + create)
  [ ] pages/agents/AgentForm.tsx      (create/edit form)
  [ ] pages/agents/AgentCard.tsx      (name, type, tools, edit/delete)
  [ ] Update ChatPage to support agent selection per conversation
```

**Build order:** simple_agent → wire into chat → test → supervisor_graph → sub-agents → tools

---

---

# Phase 6 — Voice

**Goal:** Record audio in browser → Whisper transcribes → injects as chat message.

**Time estimate:** 2–3 days

## Checklist
```
Backend
  [ ] voice/infrastructure/transcription/whisper.go
  [ ] voice/application/service/voice_service.go
  [ ] voice/delivery/http/handler.go  (multipart audio upload)
  [ ] voice/delivery/http/routes.go
  [ ] Wire in main.go

Frontend
  [ ] components/VoiceRecorder.tsx  (MediaRecorder API → upload → poll → inject message)
  [ ] Add mic button to ChatInput.tsx
```

---

---

# Phase 7 — Workflows + Tools

**Goal:** Visual multi-step automation with agent nodes, tool nodes, conditions, and scheduling.

**Time estimate:** 7–10 days

**Dependencies:**
```bash
go get github.com/robfig/cron/v3
```

## Checklist
```
Backend
  [ ] workflow/application/port/repository.go
  [ ] workflow/application/service/workflow_service.go  (CRUD)
  [ ] workflow/infrastructure/repository/workflow_repo.go
  [ ] workflow/infrastructure/repository/run_repo.go
  [ ] workflow/infrastructure/engine/dag_executor.go    (topological sort + step runner)
  [ ] workflow/infrastructure/scheduler/scheduler.go    (cron-based trigger)
  [ ] tool/infrastructure/executors/github_tool.go
  [ ] tool/infrastructure/executors/google_calendar_tool.go
  [ ] tool/infrastructure/executors/email_tool.go
  [ ] workflow/delivery/http/handler.go
  [ ] workflow/delivery/http/routes.go
  [ ] tool/delivery/http/handler.go  (list, configure)
  [ ] Wire + start scheduler in main.go

Frontend
  [ ] services/workflow.service.ts
  [ ] pages/workflows/WorkflowsPage.tsx
  [ ] pages/workflows/WorkflowBuilder.tsx  (visual DAG editor, or JSON editor for MVP)
  [ ] pages/workflows/RunHistory.tsx
```

---

---

# Phase 8 — Multi-Tenant

**Goal:** Org-level isolation. One deployment serves multiple teams.

**Time estimate:** 5–7 days

## Checklist
```
Backend
  [ ] New migration: 012_create_tenants.sql  (tenants + tenant_members tables)
  [ ] shared/middleware/tenant.go  (reads X-Tenant-ID header, validates membership)
  [ ] Update all repositories to filter by tenant_id
  [ ] Per-tenant rate limiting key: rate:{tenant_id}:{user_id}
  [ ] Tenant invite flow (invite by email)
  [ ] tenant/delivery/http/handler.go  (create, invite, list members)

Frontend
  [ ] Tenant switcher in Header
  [ ] Invite member flow
  [ ] Tenant settings page
```

---

---

# Go.mod — Add Per Phase

Only add what each phase actually uses:

```bash
# Phase 2 ✅ already added
go get github.com/sashabaranov/go-openai

# Phase 3 ✅ already added
go get github.com/minio/minio-go/v7
go get github.com/qdrant/go-client
go get google.golang.org/grpc
go get github.com/ledongthuc/pdf

# Phase 5 (when starting)
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai

# Phase 7 (when starting)
go get github.com/robfig/cron/v3

# After each: go mod tidy
```
