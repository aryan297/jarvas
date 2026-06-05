# Phase 4 — Memory: Changes & Third-Party API Guide

> What was built, what changed, and exactly where to add every API key.

---

## What Was Built

Phase 4 adds **long-term AI memory** to Jarvas. The AI now automatically extracts facts about the user after every conversation turn and injects relevant memories into the system prompt on the next message.

### Pipeline at a Glance

```
User sends a chat message
        │
        ▼
ChatService.buildSystemPrompt()
  → Qdrant semantic search for relevant memories
  → Prepend "What you know about this user:" to system prompt
        │
        ▼
OpenAI responds
        │
        ▼
bus.Publish(ChatCompleted{UserMsg, AssistMsg})
        │                          (async goroutine)
        ▼
MemoryService.Extract()
  → OpenAI: extract 0–3 facts from the exchange
  → Save each fact to Postgres (memories table)
  → Embed text → Upsert to Qdrant "memory" collection
```

---

## All Changed Files

### New Files — Backend

| File | Purpose |
|------|---------|
| `backend/internal/modules/chat/domain/event/events.go` | `ChatCompleted` domain event — fired after every assistant reply |
| `backend/internal/modules/memory/application/port/repository.go` | `MemoryRepository` interface |
| `backend/internal/modules/memory/application/dto/memory_dto.go` | Request/response DTOs |
| `backend/internal/modules/memory/application/service/memory_service.go` | Full service: CRUD, semantic search, LLM extraction, Qdrant indexing |
| `backend/internal/modules/memory/infrastructure/repository/memory_repo.go` | pgx/Postgres implementation |
| `backend/internal/modules/memory/delivery/http/handler.go` | 4 HTTP endpoints |
| `backend/internal/modules/memory/delivery/http/routes.go` | Route registration |

### Modified Files — Backend

| File | What Changed |
|------|-------------|
| `backend/internal/modules/chat/application/port/repository.go` | Added `MemoryRetriever` interface — avoids circular import between chat ↔ memory |
| `backend/internal/modules/chat/application/service/chat_service.go` | Added `SetMemoryRetriever()`, `buildSystemPrompt()` (injects memories), publishes `ChatCompleted` after every reply |
| `backend/cmd/server/main.go` | Wires memory module, calls `memoryService.EnsureCollection()`, calls `chatService.SetMemoryRetriever()`, subscribes `ChatCompleted` → `memoryService.Extract()` |

### New Files — Frontend

| File | Purpose |
|------|---------|
| `frontend/src/services/memory.service.ts` | API client: create, list, delete, search |
| `frontend/src/pages/memory/MemoryCard.tsx` | Card with type badge, importance bar, delete button |
| `frontend/src/pages/memory/MemoryPage.tsx` | Full page: tabbed list + semantic search, create form |

---

## New API Endpoints

Base URL: `http://localhost:8080/api/v1`  
All endpoints require `Authorization: Bearer <access_token>`.

### `POST /memories` — Create memory manually

```json
// Request
{
  "type": "FACT",         // FACT | PREFERENCE | EVENT | SKILL | RELATIONSHIP
  "content": "User prefers Python over Java",
  "importance": 0.7       // 0.0–1.0, defaults to 0.5
}

// Response 201
{
  "success": true,
  "data": {
    "id": "uuid",
    "type": "FACT",
    "content": "User prefers Python over Java",
    "importance": 0.7,
    "access_count": 0,
    "created_at": "2026-06-06T10:00:00Z"
  }
}
```

### `GET /memories?page=1&limit=20` — List memories

```json
// Response 200
{
  "success": true,
  "data": [...],
  "meta": { "page": 1, "limit": 20, "total": 42, "total_pages": 3 }
}
```

### `DELETE /memories/:id` — Delete memory

```
Response: 204 No Content
Side effect: also deletes the Qdrant vector
```

### `POST /memories/search` — Semantic search

```json
// Request
{ "query": "programming preferences", "top_k": 5, "min_score": 0.3 }

// Response 200
{
  "success": true,
  "data": [
    { "id": "uuid", "type": "PREFERENCE", "content": "...", "importance": 0.7, "score": 0.89 }
  ]
}
```

---

## Third-Party APIs — Where & How

### 1. OpenAI — Required for Phases 2, 3, 4+

**What it does:**
- Phase 2: Chat completions (`gpt-4o`) + streaming
- Phase 3: Text embeddings (`text-embedding-3-small`) for document RAG
- Phase 4: Memory extraction LLM call + memory query embeddings

**Where to add the key:**

```
File: .env (project root)
Variable: OPENAI_API_KEY=sk-...
```

```
File: .env.example — already has the placeholder:
  OPENAI_API_KEY=your-openai-api-key
```

**How it flows through the code:**

```
.env → config.Load() → cfg.AI.OpenAIKey
     → ChatService    (chatsvc.ChatConfig{OpenAIKey: cfg.AI.OpenAIKey})
     → MemoryService  (memsvc.NewMemoryService(..., cfg.AI.OpenAIKey, ...))
     → RAG Embedder   (ragembedding.NewOpenAIEmbedder(cfg.AI.OpenAIKey, ...))
```

**Where to get it:** [platform.openai.com/api-keys](https://platform.openai.com/api-keys)

**Model config:**

```
AI_MODEL=gpt-4o                          # Chat completions + memory extraction
AI_EMBEDDING_MODEL=text-embedding-3-small # All vector embeddings (1536 dims)
AI_EMBEDDING_DIMENSIONS=1536
```

To switch models, change these two env vars — no code changes needed.

---

### 2. Google OAuth — Required for social login

**What it does:** Lets users sign in with their Google account.

**Where to add the keys:**

```
File: .env
GOOGLE_CLIENT_ID=123456789-abc.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
```

**How to get these keys:**

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a project → APIs & Services → Credentials
3. Create OAuth 2.0 Client ID → Application type: **Web application**
4. Add Authorized redirect URI: `http://localhost:8080/api/v1/auth/google/callback`
5. Copy Client ID and Client Secret into `.env`

**For production**, add your real domain:
```
GOOGLE_REDIRECT_URL=https://api.yourdomain.com/api/v1/auth/google/callback
```

**How it flows through the code:**

```
.env → cfg.Google → oauth.NewGoogleProvider(cfg.Google)
     → authHandler.GoogleLogin() / GoogleCallback()
```

---

### 3. MinIO (S3-compatible) — Required for document uploads

**What it does:** Stores uploaded files (PDFs, docs). Works like AWS S3.

**Local setup (Docker, already in docker-compose.yml):**

```
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_BUCKET_DOCUMENTS=documents
MINIO_BUCKET_AUDIO=audio
```

**Create buckets after first `docker compose up`:**

```bash
# Option 1 — MinIO web console
open http://localhost:9001  # login: minioadmin / minioadmin
# Create buckets: "documents" and "audio"

# Option 2 — mc CLI
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/documents
mc mb local/audio
```

**For production (AWS S3):** Replace MinIO with real S3 credentials:

```
MINIO_ENDPOINT=s3.amazonaws.com
MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
MINIO_USE_SSL=true
```

The `docstorage.NewMinIOStorage()` constructor uses the same minio-go SDK for both — no code changes needed.

---

### 4. Qdrant — Required for RAG + Memory (Phase 3+)

**What it does:** Vector database for semantic search. Phase 3 uses collection `"documents"`, Phase 4 adds collection `"memory"`.

**Local setup (Docker, already in docker-compose.yml):**

```
QDRANT_HOST=localhost
QDRANT_PORT=6334          # gRPC port (6333 is HTTP dashboard)
QDRANT_API_KEY=           # empty for local dev
QDRANT_COLLECTION_DOCUMENTS=documents
QDRANT_COLLECTION_MEMORY=memory
```

**Collections are auto-created on startup** — `processor.EnsureCollection()` and `memoryService.EnsureCollection()` in `main.go` handle this. You don't create them manually.

**For production (Qdrant Cloud):**

```
QDRANT_HOST=your-cluster-id.us-east4-0.gcp.cloud.qdrant.io
QDRANT_PORT=6334
QDRANT_API_KEY=your-qdrant-api-key
```

Get a key at [cloud.qdrant.io](https://cloud.qdrant.io).

---

### 5. Anthropic (Claude) — Future / Phase 5+

Not used yet but wired into config for Phase 5 (Eino agents):

```
ANTHROPIC_API_KEY=sk-ant-...
```

**Where to get it:** [console.anthropic.com/keys](https://console.anthropic.com/keys)

**Where it will be used:** `agent/infrastructure/eino/` — Eino supports multiple LLM backends including Anthropic.

---

## Complete `.env` Reference

Annotated with which phase each variable is first needed:

```bash
# ── App ───────────────────────────────────────────────────────────────────────
APP_ENV=development           # development | production
APP_PORT=8080
APP_NAME=jarvas

# ── Database (required from Phase 1) ─────────────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USER=jarvas
DB_PASSWORD=jarvas_secret     # REQUIRED — set to any strong password
DB_NAME=jarvas_db
DB_SSL_MODE=disable           # use "require" in production
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10

# ── Redis (required from Phase 1) ─────────────────────────────────────────────
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=               # set in production
REDIS_DB=0

# ── JWT (required from Phase 1) ───────────────────────────────────────────────
JWT_SECRET=change-me-32-chars-minimum    # REQUIRED — openssl rand -hex 32
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h       # 7 days

# ── Google OAuth (required from Phase 1) ─────────────────────────────────────
GOOGLE_CLIENT_ID=             # REQUIRED — from Google Cloud Console
GOOGLE_CLIENT_SECRET=         # REQUIRED — from Google Cloud Console
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# ── OpenAI (required from Phase 2) ───────────────────────────────────────────
OPENAI_API_KEY=sk-...         # REQUIRED for chat, embeddings, memory extraction
AI_MODEL=gpt-4o               # Chat completions + memory extraction
AI_EMBEDDING_MODEL=text-embedding-3-small
AI_EMBEDDING_DIMENSIONS=1536

# ── MinIO / S3 (required from Phase 3) ───────────────────────────────────────
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_BUCKET_DOCUMENTS=documents
MINIO_BUCKET_AUDIO=audio      # used in Phase 6 (Voice)

# ── Qdrant (required from Phase 3) ────────────────────────────────────────────
QDRANT_HOST=localhost
QDRANT_PORT=6334
QDRANT_API_KEY=               # required for Qdrant Cloud
QDRANT_COLLECTION_DOCUMENTS=documents
QDRANT_COLLECTION_MEMORY=memory   # added Phase 4

# ── Anthropic (Phase 5 — AI Agents) ──────────────────────────────────────────
ANTHROPIC_API_KEY=sk-ant-...  # optional until Phase 5

# ── CORS ──────────────────────────────────────────────────────────────────────
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
```

---

## Architecture Note — Import Cycle Solution

Memory and Chat are peer modules. Chat needs to call into Memory (to fetch relevant memories), and Memory needs to subscribe to ChatCompleted (fired by Chat). This would normally create a circular import.

**Solution used:**

```
chat/application/port/repository.go
    defines MemoryRetriever interface
           ↑
           │ implements (implicitly — Go duck typing)
           │
memory/application/service/memory_service.go
    has SearchRelevant(ctx, userID, query, limit) ([]string, error)
           │
           │ injected by
           ▼
cmd/server/main.go
    chatService.SetMemoryRetriever(memoryService)
```

- `chat` never imports `memory`
- `memory` never imports `chat`
- `main.go` is the only place that knows both exist and wires them together

---

## How to Test Phase 4

```bash
TOKEN="your-bearer-token"

# 1. Send a few chat messages — memories auto-extract in the background
curl -X POST :8080/api/v1/conversations/YOUR_CONV_ID/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "I prefer TypeScript over plain JavaScript and I work at Acme Corp"}'

# 2. Wait ~2 seconds for async extraction, then list memories
curl :8080/api/v1/memories \
  -H "Authorization: Bearer $TOKEN"

# 3. Create a memory manually
curl -X POST :8080/api/v1/memories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"SKILL","content":"User is proficient in Go","importance":0.8}'

# 4. Semantic search
curl -X POST :8080/api/v1/memories/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"programming languages","top_k":5}'

# 5. Delete a memory
curl -X DELETE :8080/api/v1/memories/MEMORY_ID \
  -H "Authorization: Bearer $TOKEN"

# 6. Confirm memory is injected into next chat reply
# In the system prompt you'll see "## What you know about this user:" section
```

---

## Phase Status After This PR

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | Auth | ✅ Done |
| 2 | Chat + Streaming | ✅ Done |
| 3 | Documents + RAG | ✅ Done |
| 4 | Memory | ✅ Done |
| 5 | AI Agents (Eino) | Next |
| 6 | Voice | Pending |
| 7 | Workflows + Tools | Pending |
| 8 | Multi-Tenant | Pending |
