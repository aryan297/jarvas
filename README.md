# Jarvas — Production-Grade AI Agent Platform

A multi-tenant AI assistant platform built for scale: 0 → 5k–10k users as a
Modular Monolith, with a clear extraction path to microservices.

---

## Build Status

| Phase | Feature            | Status        |
|-------|--------------------|---------------|
| 1     | Auth               | ✅ **Done**   |
| 2     | Chat + Streaming   | ✅ **Done**   |
| 3     | Documents + RAG    | ✅ **Done**   |
| 4     | Memory             | ✅ **Done**   |
| 5     | AI Agents (Eino)   | ✅ **Done**   |
| 6     | Voice              | ✅ **Done**   |
| 7     | Workflows + Tools  | ✅ **Done**   |
| 8     | Multi-Tenant       | ✅ **Done**   |

**130 backend module files · 49 frontend files · 0 build errors**

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
├── chat/      ✅  Conversations, messages, SSE streaming, Redis short-term memory
├── document/  ✅  Upload → MinIO → async RAG indexing pipeline
├── rag/       ✅  Chunk → embed → Qdrant → rerank → context string
├── memory/    ✅  Long-term memory: Postgres + Qdrant, LLM auto-extraction
├── agent/     ✅  Eino ChatModel, tool calling loop, memory injection, 5 endpoints
├── voice/     ✅  Audio upload → Whisper → transcript injected into chat input
├── workflow/  ✅  DAG executor, cron scheduler, tool nodes, run history
├── tool/      ✅  Registry (web_search, calculator, http_request) + DB-backed tool API
└── tenant/    ✅  Orgs, membership (OWNER/ADMIN/MEMBER), invite by email, workspace switcher
```

Every module: `domain/ → application/ → infrastructure/ → delivery/`

### Agent Pipeline (Phase 5 — Live)

```
POST /conversations  { agent_id: "uuid" }
  → conversation pinned to agent

POST /conversations/:id/messages  { content: "..." }
  → ChatService detects conv.agent_id → RunnerService.RunMessage()
  → Fetch agent config (model, temp, tools, system_prompt)
  → Memory injection if agent.memory_enabled
  → EinoRunner: bind tools → Generate loop → tool calls executed → final reply
  → ChatCompleted event → async memory extraction
```

### Memory Pipeline (Phase 4 — Live)

```
User sends chat message
  → Qdrant: fetch top-5 relevant long-term memories
  → Prepend to system prompt: "What you know about this user: ..."
  → OpenAI generates response
  → bus.Publish(ChatCompleted{userMsg, assistMsg})   ← async

Background goroutine:
  → OpenAI: extract 0–3 facts/preferences/skills from exchange
  → INSERT into memories (Postgres)
  → EmbedText → Upsert to Qdrant "memory" collection
```

### RAG Pipeline (Phase 3 — Live)

```
POST /documents (multipart)
  → MinIO upload
  → bus.Publish(DocumentUploaded)   ← responds 201 here
  → background goroutine:
      → extract text (PDF / TXT / MD / CSV / HTML)
      → chunk (512 tokens, 50 overlap)
      → embed batch (text-embedding-3-small)
      → upsert Qdrant (payload: user_id for tenant isolation)
      → update status=INDEXED

POST /rag/search
  → embed query
  → Qdrant ANN (filtered by user_id)
  → rerank top-5
  → return ranked chunks with source doc name
```

### Internal Event Bus

Modules communicate via events — no direct calls across module boundaries.

| Event                  | Publisher | Subscriber              |
|------------------------|-----------|-------------------------|
| `auth.user_registered` | auth      | audit, billing (future) |
| `document.uploaded`    | document  | rag processor           |
| `document.indexed`     | rag       | audit                   |
| `document.deleted`     | document  | qdrant cleanup          |
| `chat.completed`       | chat      | **memory extractor** ✅ |
| `memory.created`       | memory    | audit (future)          |
| `workflow.executed`    | workflow  | audit, billing (future) |

---

## Quick Start

```bash
# 1. Clone and configure
git clone <repo> && cd jarvas
cp .env.example .env
# Required always:   JWT_SECRET, DB_PASSWORD
# Required Phase 2+: OPENAI_API_KEY
# Required Phase 1:  GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET

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
                                              body.stream=true → SSE streaming
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
POST  /api/v1/rag/search             Semantic search {query, top_k, min_score}
```

### Memory ✅
```
POST  /api/v1/memories               Create memory manually
GET   /api/v1/memories               List memories (sorted by importance)
DELETE /api/v1/memories/:id          Delete memory + Qdrant vector
POST  /api/v1/memories/search        Semantic search {query, top_k, min_score}
```

### Voice ✅
```
POST /api/v1/voice/upload           Multipart: audio + conversation_id → 202 { session_id }
GET  /api/v1/voice/sessions/:id     Poll for transcript { status, transcript }
GET  /api/v1/voice/sessions         List recent sessions
```

### Tenants ✅
```
POST   /api/v1/tenants                    Create workspace (caller = OWNER)
GET    /api/v1/tenants                    List user's workspaces
GET    /api/v1/tenants/:id                Get workspace
POST   /api/v1/tenants/:id/invite         Invite by email {email, role}
GET    /api/v1/tenants/:id/members        List members
DELETE /api/v1/tenants/:id/members/:uid   Remove member
```

### Workflows ✅
```
POST  /api/v1/workflows              Create workflow (definition: nodes+edges+trigger)
GET   /api/v1/workflows              List workflows
GET   /api/v1/workflows/:id          Get workflow
PATCH /api/v1/workflows/:id          Update (status, definition, cron)
DELETE /api/v1/workflows/:id         Archive
POST  /api/v1/workflows/:id/run      Trigger run → 202 { run_id }
GET   /api/v1/workflows/:id/runs     Run history (paginated)
GET   /api/v1/workflows/:id/runs/:run_id  Poll run status

GET   /api/v1/tools                  List all tools (5 built-in seeded)
POST  /api/v1/tools/:name/configure  Save user credentials for a tool
GET   /api/v1/tools/:name/config     Get user's tool config
```

### Agents ✅
```
POST  /api/v1/agents          Create agent {name, type, model, system_prompt, tools_enabled, ...}
GET   /api/v1/agents          List agents (paginated)
GET   /api/v1/agents/:id      Get agent
PATCH /api/v1/agents/:id      Update agent
DELETE /api/v1/agents/:id     Soft-delete (is_active=false)

# To use an agent, create a conversation with agent_id then send messages normally:
POST  /api/v1/conversations   { "agent_id": "<uuid>" }
POST  /api/v1/conversations/:id/messages  { "content": "..." }
                              → chat is routed through the Eino runner
```

---

## Environment Variables

Minimum required to run each phase:

| Variable | Phase | Description |
|----------|-------|-------------|
| `DB_PASSWORD` | 1 | Postgres password |
| `JWT_SECRET` | 1 | Min 32 chars (`openssl rand -hex 32`) |
| `GOOGLE_CLIENT_ID` | 1 | Google Cloud Console → OAuth 2.0 |
| `GOOGLE_CLIENT_SECRET` | 1 | Google Cloud Console → OAuth 2.0 |
| `OPENAI_API_KEY` | 2+ | Chat completions, embeddings, memory extraction |
| `QDRANT_API_KEY` | 3+ | Required only for Qdrant Cloud (empty for local) |
| `ANTHROPIC_API_KEY` | 5 | Claude — used in Eino agents (Phase 5) |

See `.env.example` for all variables with defaults, or [PHASE4_CHANGES.md](PHASE4_CHANGES.md) for a full annotated reference.

---

## Security

- **JWT**: HS256, 15-min access token, 7-day rotating refresh (SHA-256 hashed in DB)
- **Qdrant isolation**: every search filters `payload.user_id` — users never see each other's data
- **Memory isolation**: same pattern — all memory Qdrant searches filter by `user_id`
- **OAuth CSRF**: state stored in Redis (10 min TTL), deleted on use
- **Passwords**: bcrypt cost 10
- **Presigned URLs**: MinIO 24h URLs — backend never proxies file downloads
- **Error logging**: 5xx errors log the cause server-side; clients only see a generic message

---

## Documentation

| File | Contents |
|------|----------|
| [JARVAS.md](JARVAS.md) | Full developer guide: architecture, all APIs, env vars, coding conventions |
| [PHASES.md](PHASES.md) | Phase-by-phase build checklist with exact files, code, and curl tests |
| [PHASE4_CHANGES.md](PHASE4_CHANGES.md) | Phase 4 change log: all new/modified files + third-party API setup guide |

---

## Microservice Extraction Order

When a module needs to scale independently:

1. `document` + `rag` — I/O and CPU heavy, extract first
2. `voice` — Whisper needs GPU nodes
3. `workflow` — stateful, long-running, add Temporal
4. `agent` — compute-heavy, independent replicas
5. `billing` — compliance boundary

Each extraction: move module folder → swap `bus.Publish()` for NATS/Kafka → swap function calls for gRPC.
