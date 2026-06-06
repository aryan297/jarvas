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
| 4     | Memory            | ✅ Complete   | ✅ Complete   | **DONE**         |
| 5     | AI Agents (Eino)  | ✅ Complete   | ✅ Complete   | **DONE**         |
| 6     | Voice             | 🔲 Not started | 🔲 Not started | **▶ ACTIVE NEXT**|
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

# ✅ Phase 4 — Memory (COMPLETE)

**Goal:** AI remembers facts across conversations. Short-term (Redis), long-term (Postgres + Qdrant).

## Files Built

```
Backend
  ✅ chat/domain/event/events.go                     ChatCompleted event
  ✅ chat/application/port/repository.go              MemoryRetriever interface (avoids import cycle)
  ✅ chat/application/service/chat_service.go         Memory injection + ChatCompleted publish
  ✅ memory/application/port/repository.go            MemoryRepository interface
  ✅ memory/application/dto/memory_dto.go             Create/search/response DTOs
  ✅ memory/application/service/memory_service.go     CRUD + semantic search + LLM extraction
  ✅ memory/infrastructure/repository/memory_repo.go  pgx implementation
  ✅ memory/delivery/http/handler.go                  4 endpoints
  ✅ memory/delivery/http/routes.go

Frontend
  ✅ services/memory.service.ts
  ✅ pages/memory/MemoryPage.tsx   (tabbed: list + semantic search, create, delete)
  ✅ pages/memory/MemoryCard.tsx   (type badge, importance bar, content)
```

## Verified Endpoints

```
POST   /api/v1/memories           Create memory manually
GET    /api/v1/memories           List user memories (paginated, sorted by importance)
DELETE /api/v1/memories/:id       Delete memory + Qdrant vector
POST   /api/v1/memories/search    Semantic search (body: {query, top_k, min_score})
```

## Pipeline Flow

```
Chat message sent
  → ChatService.SendMessage / StreamMessage
  → Retrieve top-5 relevant memories → append to system prompt
  → Call OpenAI with enriched context
  → bus.Publish(ChatCompleted{UserID, UserMsg, AssistMsg})

Background goroutine (ChatCompleted handler):
  → MemoryService.Extract(userID, userMsg, assistMsg)
  → OpenAI call: extract 0–3 facts/preferences/skills about the user
  → Parse JSON response
  → Save each memory to Postgres
  → EmbedText → Upsert to Qdrant "memory" collection

DELETE /memories/:id
  → DeleteByFilter(Qdrant, {memory_id: id})
  → DELETE from Postgres
```

## Key Design Decisions

- **Import cycle avoided**: `MemoryRetriever` interface defined in `chat/application/port` — memory module never imports chat
- **Graceful degradation**: `memSvc` is injected via `SetMemoryRetriever()` after construction; if nil, chat works normally
- **Separate Qdrant collection**: "memory" collection is isolated from "documents" — same QdrantStore client, different collection name
- **Auto-extraction**: every chat turn triggers async LLM extraction — user never has to manage memories manually
- **Tenant isolation**: all Qdrant searches filter by `user_id` payload field

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

# ✅ Phase 5 — AI Agents (Eino) (COMPLETE)

**Goal:** Named agents with custom prompts, tool calling, memory + RAG injection, wired into chat.

**Dependencies added:**
```bash
go get github.com/cloudwego/eino@v0.9.4
go get github.com/cloudwego/eino-ext/components/model/openai@v0.1.13
```

## Files Built

```
Backend
  ✅ agent/application/port/repository.go               AgentRepository interface
  ✅ agent/application/service/agent_service.go          CRUD: Create, List, GetByID, Update, Delete
  ✅ agent/application/service/runner_service.go         Implements AgentRunnerPort; injects memory
  ✅ agent/infrastructure/repository/agent_repo.go       pgx — JSONB tools_enabled, agent_type cast
  ✅ agent/infrastructure/eino/runner.go                 Eino ChatModel + tool calling loop (max 5 iters)
  ✅ tool/application/service/tool_registry.go           Registry: name → ToolDef{Info, Execute}
  ✅ tool/infrastructure/executors/web_search.go         DuckDuckGo instant answers (no API key)
  ✅ tool/infrastructure/executors/calculator.go         Arithmetic: add/subtract/multiply/divide/modulo
  ✅ agent/delivery/http/handler.go                      5 endpoints (CRUD)
  ✅ agent/delivery/http/routes.go

Frontend
  ✅ services/agent.service.ts                          create, list, getById, update, delete
  ✅ pages/agents/AgentsPage.tsx                        List + create/edit/delete + "Chat with agent" button
  ✅ pages/agents/AgentForm.tsx                         Full form: type, model, temp, tools, memory, RAG
  ✅ pages/agents/AgentCard.tsx                         Type badge, tool chips, meta row
  ✅ pages/chat/ChatPage.tsx                            Updated: reads ?agent_id param, shows agent selector
```

## Verified Endpoints

```
POST  /api/v1/agents          Create agent
GET   /api/v1/agents          List agents (paginated)
GET   /api/v1/agents/:id      Get agent
PATCH /api/v1/agents/:id      Update agent
DELETE /api/v1/agents/:id     Soft-delete (is_active=false)
```

## Tool Calling Flow

```
User sends message to agent conversation
        │
        ▼
ChatService detects conv.agent_id → delegates to RunnerService.RunMessage
        │
        ▼
RunnerService.buildRunConfig()
  → Fetch agent from DB (model, temp, max_tokens, tools, system_prompt)
  → If memory_enabled: SearchRelevant → prepend to system prompt
  → Look up enabled tools in ToolRegistry → get []schema.ToolInfo
        │
        ▼
EinoRunner.Run()
  → openai.NewChatModel(apiKey, model, temp, maxTokens)
  → cm.BindTools(toolInfos)
  → Loop up to 5 iterations:
      - cm.Generate(msgs) → resp
      - if resp.ToolCalls == nil → return resp.Content
      - for each tool call: registry.Execute(name, argsJSON) → result
      - append ToolMessage(result, callID) to msgs
  → return final assistant text
```

## Key Design Decisions

- **AgentRunnerPort in chat/port**: avoids import cycle between chat ↔ agent (same pattern as MemoryRetriever)
- **SetAgentRunner()** injected after construction in main.go
- **Tool registry**: stateless — any function `(argsJSON string) → (string, error)` can be registered
- **Built-in tools**: `web_search` (DuckDuckGo, no key) + `calculator` (safe structured arithmetic)
- **Memory in agents**: RunnerService has its own `MemoryRetriever` so agents get user context too
- **Streaming agents**: tool loop uses `Generate()`, final response emitted word-by-word to simulate streaming

---

---

# ✅ Phase 6 — Voice (COMPLETE)

**Goal:** Record audio in browser → Whisper transcribes → injects as chat message.

## Files Built

```
Backend
  ✅ voice/application/port/ports.go                 SessionStore, AudioStore, TranscriptionPort interfaces
  ✅ voice/application/dto/voice_dto.go               Upload request/response + VoiceSessionResponse
  ✅ voice/application/service/voice_service.go       Upload (async), GetSession, ListSessions
  ✅ voice/infrastructure/transcription/whisper.go    OpenAI Whisper API (go-openai SDK)
  ✅ voice/infrastructure/storage/audio_storage.go    MinIO audio bucket (audio/{user_id}/{session_id}.ext)
  ✅ voice/infrastructure/cache/session_store.go      Redis session store (24h TTL, no migration needed)
  ✅ voice/delivery/http/handler.go                   3 endpoints
  ✅ voice/delivery/http/routes.go

Frontend
  ✅ services/voice.service.ts                        upload, getSession, listSessions, pollUntilDone
  ✅ components/VoiceRecorder.tsx                     MediaRecorder → upload → poll → inject transcript
  ✅ pages/chat/ChatInput.tsx                         Mic button added; transcript injected into textarea
  ✅ pages/chat/MessageThread.tsx                     Passes convId down to ChatInput
```

## Verified Endpoints

```
POST /api/v1/voice/upload           Multipart: audio (file) + conversation_id + language?
                                    → 202 Accepted { session_id, status: "PENDING" }
GET  /api/v1/voice/sessions/:id     Poll: { status, transcript, duration_seconds, ... }
GET  /api/v1/voice/sessions         List user's recent sessions (last 20)
```

## Flow

```
User clicks mic → MediaRecorder starts recording
User clicks stop → Blob created (audio/webm)
  → POST /voice/upload (multipart)
  → MinIO: audio/{user_id}/{session_id}.webm
  → Redis: VoiceSession{status=PENDING}
  → Background goroutine starts
  → Returns 202 with session_id

Frontend polls GET /voice/sessions/:id every 2s (max 30s):
  Background goroutine:
    → Redis: status=PROCESSING
    → Download from MinIO
    → OpenAI Whisper API → transcript text
    → Redis: status=COMPLETED, transcript="..."

Frontend sees COMPLETED → injects transcript into ChatInput textarea
User reviews text → presses Enter to send as a chat message
```

## Key Design Decisions

- **No DB migration**: voice sessions stored in Redis (24h TTL) — ephemeral by design
- **Async upload**: returns 202 immediately; Whisper runs in goroutine — no request timeout
- **Transcript injection**: text is placed in the textarea for user review, NOT auto-sent
- **Supported formats**: webm, mp4, mpeg, ogg, wav, m4a (25 MB Whisper limit enforced server-side)
- **MinIO bucket**: uses existing `MINIO_BUCKET_AUDIO` (env default: "audio") — bucket must be created on first run

---

---

# ✅ Phase 7 — Workflows + Tools (COMPLETE)

**Goal:** Multi-step automation with AI agent nodes, tool nodes, conditions, scheduling, and run history.

**Dependency added:** `go get github.com/robfig/cron/v3`

## Files Built

```
Backend
  ✅ workflow/application/port/repository.go              WorkflowRepository, RunRepository
  ✅ workflow/application/dto/workflow_dto.go              Create/Update/Trigger DTOs
  ✅ workflow/application/service/workflow_service.go      CRUD + TriggerRun + ListRuns + GetRun
  ✅ workflow/infrastructure/repository/workflow_repo.go   pgx — JSONB definition, enum casts
  ✅ workflow/infrastructure/repository/run_repo.go        pgx
  ✅ workflow/infrastructure/engine/dag_executor.go        Topological sort (Kahn's) + step runner
  ✅ workflow/infrastructure/scheduler/scheduler.go        robfig/cron wrapper — loads ACTIVE+SCHEDULE on start
  ✅ workflow/delivery/http/handler.go                     8 endpoints
  ✅ workflow/delivery/http/routes.go

  ✅ tool/application/port/repository.go                  ToolRepository, UserToolConfigRepository
  ✅ tool/infrastructure/repository/tool_repo.go           pgx — reads seeded tools + user configs
  ✅ tool/infrastructure/executors/http_request.go         Generic HTTP request tool (headers, body, method)
  ✅ tool/delivery/http/handler.go                         List tools + configure per-user credentials
  ✅ tool/delivery/http/routes.go

Frontend
  ✅ services/workflow.service.ts                          CRUD, triggerRun, listRuns, getRun
  ✅ services/tool.service.ts                              list, getConfig, configure
  ✅ pages/workflows/WorkflowsPage.tsx                     List + create + run + delete; collapsible run history
  ✅ pages/workflows/WorkflowBuilder.tsx                   Visual step builder + JSON editor toggle
  ✅ pages/workflows/RunHistory.tsx                        Status badges, duration, auto-polls active runs
```

## Verified Endpoints

```
POST  /api/v1/workflows              Create workflow
GET   /api/v1/workflows              List (paginated)
GET   /api/v1/workflows/:id          Get workflow + definition
PATCH /api/v1/workflows/:id          Update (name, status, definition, cron)
DELETE /api/v1/workflows/:id         Soft-delete (status=ARCHIVED)
POST  /api/v1/workflows/:id/run      Trigger manual run → 202 Accepted { run_id }
GET   /api/v1/workflows/:id/runs     List runs (paginated)
GET   /api/v1/workflows/:id/runs/:run_id  Get run (poll for status)

GET   /api/v1/tools                  List all registered tools
GET   /api/v1/tools/:name/config     Get user's saved config for a tool
POST  /api/v1/tools/:name/configure  Save/update user credentials for a tool
```

## DAG Execution Flow

```
POST /workflows/:id/run
  → Create WorkflowRun (PENDING) in DB
  → Return 202 immediately
  → Background goroutine: DAGExecutor.Execute()
      → Kahn's topological sort on nodes + edges
      → For each node in order:
          agent node:     OpenAI chat completion with {template} var substitution
          tool node:      ToolRegistry.Execute(name, argsJSON)
          condition node: evaluate {expr} → route "then" or "else"
          delay node:     time.Sleep(N seconds, max 300)
          → execCtx[node_id+"_output"] = result
      → Update run: COMPLETED + result map / FAILED + error_msg

Scheduler (startup):
  → Load all ACTIVE+SCHEDULE workflows
  → Register each cron job (robfig/cron)
  → On fire: TriggerRun with {trigger:"schedule"} payload
```

## Key Design Decisions

- **No import cycle**: `DAGExecutor` interface defined in `workflow/application/service` — engine implements it
- **Variable substitution**: `{node_id_output}` in any string field is replaced before execution
- **Tool registry reuse**: same Phase 5 registry (web_search, calculator) + new http_request
- **Tool DB seeded**: migration 010 pre-seeds 5 built-in tools; users add credentials via `/tools/:name/configure`
- **Async execution**: always returns 202 + run_id; client polls `/runs/:id` until COMPLETED/FAILED
- **Scheduler**: starts on server boot; loaded from DB — survives restarts

---

---

# ✅ Phase 8 — Multi-Tenant (COMPLETE)

**Goal:** Org-level isolation. One deployment serves multiple teams.

## Files Built

```
Backend
  ✅ migrations/012_create_tenants.sql               tenants + tenant_members (roles: OWNER/ADMIN/MEMBER)
  ✅ tenant/domain/entity/tenant.go                  Tenant, TenantMember entities
  ✅ tenant/application/port/repository.go            TenantRepository, MemberRepository interfaces
  ✅ tenant/application/dto/tenant_dto.go             Create/Invite DTOs + responses
  ✅ tenant/application/service/tenant_service.go     CRUD + InviteMember + ListMembers + RemoveMember
  ✅ tenant/infrastructure/repository/tenant_repo.go  pgx — tenant + member repos + FindUserByEmail
  ✅ tenant/delivery/http/handler.go                  6 endpoints
  ✅ tenant/delivery/http/routes.go
  ✅ shared/middleware/tenant.go                      X-Tenant-ID validation (opt-in, non-breaking)

Frontend
  ✅ services/tenant.service.ts                       create, list, listMembers, invite, removeMember
  ✅ store/tenantStore.ts                             Zustand: tenants list + activeTenantId (persisted)
  ✅ components/layout/Header.tsx                     Workspace switcher dropdown
  ✅ pages/settings/SettingsPage.tsx                  Workspaces tab + Members tab with invite form
```

## Verified Endpoints

```
POST   /api/v1/tenants                        Create tenant (caller becomes OWNER)
GET    /api/v1/tenants                        List all tenants caller belongs to
GET    /api/v1/tenants/:id                    Get tenant (membership required)
POST   /api/v1/tenants/:id/invite             Invite by email {email, role}
GET    /api/v1/tenants/:id/members            List members with name + email
DELETE /api/v1/tenants/:id/members/:user_id   Remove member (OWNER/ADMIN only)
```

## Key Design Decisions

- **Personal tenant auto-created**: on UserRegistered event → personal workspace created for every new user
- **Non-breaking**: X-Tenant-ID header is optional — existing routes still work user-scoped without it
- **Middleware position**: ValidateTenant is a Gin middleware function — applied per-route-group, not globally
- **Invite by email**: requires user to already have a Jarvas account (no email sending needed for MVP)
- **Role hierarchy**: OWNER cannot be removed; MEMBER cannot invite/remove; ADMIN can manage members
- **Slug generation**: `slugify(name) + "-" + uuid[:8]` guarantees uniqueness without a DB round-trip
- **No repo rewrites**: existing data stays user-scoped; tenant layer adds collaboration on top

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
