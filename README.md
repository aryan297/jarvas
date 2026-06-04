# Jarvas — Production-Grade AI Agent Platform

A multi-tenant AI assistant platform built for scale: 0 → 5k–10k users as a
Modular Monolith, with a clear extraction path to microservices.

---

## Build Status

| Phase | Feature            | Status       |
|-------|--------------------|--------------|
| 1     | Auth               | ✅ **Done**  |
| 2     | Chat + Streaming   | ✅ **Done**  |
| 3     | Documents + RAG    | ✅ **Done**  |
| 4     | Memory             | ▶ **Next**  |
| 5     | AI Agents (Eino)   | 🔲 Pending  |
| 6     | Voice              | 🔲 Pending  |
| 7     | Workflows + Tools  | 🔲 Pending  |
| 8     | Multi-Tenant       | 🔲 Pending  |

**48 backend module files · 30 frontend files · 0 build errors**

---

## Tech Stack

| Layer        | Technology               | Why                                                        |
|--------------|--------------------------|------------------------------------------------------------|
| Backend      | Go 1.24 + Gin            | ~10x lower memory than Node. Goroutines are free.          |
| AI           | Eino (CloudWeGo)         | Production Go LLM orchestration, graph-based agents        |
| Database     | PostgreSQL 16            | ACID, JSONB, partitioning, full-text search                |
| Cache        | Redis 7                  | Short-term memory, refresh token store, OAuth CSRF state   |
| Vector DB    | Qdrant                   | Fast ANN search, payload filtering for tenant isolation    |
| Object Store | MinIO                    | S3-compatible, presigned URLs, self-hosted                 |
| Auth         | JWT + Google OAuth       | Stateless access (15 min) + rotating refresh (7 day)       |
| Frontend     | React 18 + Vite + TS     | Instant HMR, type-safe, Tailwind                           |
| Deploy       | Docker Compose → K8s     | Start simple, extract modules when traffic demands it      |

---

## Architecture

### Modular Monolith + Clean Architecture

```
backend/internal/modules/
├── auth/      ✅  JWT, Google OAuth, RBAC, refresh token rotation
├── chat/      ✅  Conversations, messages, SSE streaming, Redis memory
├── document/  ✅  Upload → MinIO → async RAG indexing pipeline
├── rag/       ✅  Chunk → embed → Qdrant → rerank → context string
├── memory/    🔲  Short-term (Redis) + long-term (Postgres + Qdrant)
├── agent/     🔲  Eino graph: supervisor + research/coding/planning
├── voice/     🔲  Audio → Whisper transcription → chat injection
├── workflow/  🔲  DAG engine, cron triggers, tool execution
└── tool/      🔲  Pluggable tool registry (HTTP, DB, GitHub, Calendar)
```

Every module: `domain/ → application/ → infrastructure/ → delivery/`

### RAG Pipeline (Live)

```
POST /documents (multipart)
  → MinIO upload
  → bus.Publish(DocumentUploaded)   ← responds 201 here
  → background goroutine
      → extract text (PDF/TXT/MD/CSV)
      → chunk (512 tokens, 50 overlap)
      → embed batch (text-embedding-3-small)
      → upsert Qdrant (payload: user_id for isolation)
      → update status=INDEXED

POST /rag/search
  → embed query
  → Qdrant ANN (filtered by user_id)
  → rerank top-5
  → return ranked chunks with source doc name
```

### Internal Event Bus

Modules communicate via events — no direct calls across module boundaries.

| Event                     | Publisher | Subscriber             |
|---------------------------|-----------|------------------------|
| `auth.user_registered`    | auth      | audit, billing (future)|
| `document.uploaded`       | document  | rag processor          |
| `document.indexed`        | rag       | audit                  |
| `document.deleted`        | document  | qdrant cleanup         |
| `chat.completed`          | chat      | memory extractor (P4)  |
| `memory.created`          | memory    | audit (future)         |
| `workflow.executed`       | workflow  | audit, billing (future)|

---

## Quick Start

```bash
# 1. Clone and configure
git clone <repo> && cd jarvas
cp .env.example .env
# Required: JWT_SECRET, DB_PASSWORD
# For chat+RAG: OPENAI_API_KEY
# For OAuth:   GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET

# 2. First-time setup (starts Docker, migrates DB, installs npm deps)
make setup

# 3. Run (two terminals)
make dev          # Go backend  :8080  (auto-starts Docker infra)
make dev-front    # React       :5173

# Or run everything in Docker
make docker-up
```

---

## Live API Endpoints

All routes require `Authorization: Bearer <access_token>` except auth endpoints.

### Auth ✅
```
POST  /api/v1/auth/register           Register (email/password)
POST  /api/v1/auth/login              Login → access + refresh token
POST  /api/v1/auth/refresh            Rotate refresh token (cookie or body)
POST  /api/v1/auth/logout             Revoke refresh token    [protected]
GET   /api/v1/auth/google/login       Get Google OAuth URL
GET   /api/v1/auth/google/callback    OAuth callback
GET   /api/v1/auth/me                 Current user profile    [protected]
```

### Chat ✅
```
POST  /api/v1/conversations                   Create conversation
GET   /api/v1/conversations                   List (paginated)
GET   /api/v1/conversations/:id               Get + messages
DELETE /api/v1/conversations/:id              Archive
POST  /api/v1/conversations/:id/messages      Send message
                                              body.stream=true → SSE
GET   /api/v1/conversations/:id/messages      List messages (paginated)
```

### Documents + RAG ✅
```
POST  /api/v1/documents              Upload (multipart/form-data, field: file)
GET   /api/v1/documents              List documents (paginated)
GET   /api/v1/documents/:id          Get + status (UPLOADED/PROCESSING/INDEXED/FAILED)
DELETE /api/v1/documents/:id         Delete + Qdrant cleanup
GET   /api/v1/documents/:id/url      24h presigned download URL
GET   /api/v1/documents/:id/chunks   List raw text chunks
POST  /api/v1/rag/search             Semantic search (body: {query, top_k, min_score})
```

### Memory (Phase 4 — coming next)
```
GET   /api/v1/memories               List memories
POST  /api/v1/memories               Create memory
PATCH /api/v1/memories/:id           Update
DELETE /api/v1/memories/:id          Delete
POST  /api/v1/memories/search        Semantic search
```

---

## Security

- **JWT**: HS256, 15-min access token, 7-day rotating refresh (SHA-256 hashed in DB)
- **Qdrant isolation**: every search filters `payload.user_id` — users never see each other's data
- **OAuth CSRF**: state stored in Redis (10 min TTL), deleted on use
- **Passwords**: bcrypt cost 10
- **Presigned URLs**: MinIO 24h URLs — backend never proxies file downloads
- **Error logging**: 5xx errors log the cause server-side; clients only see generic message

---

## Documentation

| File | Contents |
|------|----------|
| [JARVAS.md](JARVAS.md) | Full developer guide: architecture, all APIs, env vars, coding conventions |
| [PHASES.md](PHASES.md) | Phase-by-phase build checklist with exact files, code, and curl tests |

---

## Microservice Extraction Order

When a module needs to scale independently:

1. `document` + `rag` — I/O and GPU heavy, extract first
2. `voice` — Whisper needs GPU nodes
3. `workflow` — stateful long-running, add Temporal
4. `agent` — compute-heavy, independent replicas
5. `billing` — compliance boundary

Each extraction: move module folder → swap `bus.Publish()` for NATS/Kafka → swap function calls for gRPC.
