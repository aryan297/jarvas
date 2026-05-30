# Jarvas — Production-Grade AI Agent Platform

A multi-tenant AI assistant platform built for scale: 0 → 5k–10k users as a
Modular Monolith, with a clear extraction path to microservices.

---

## Tech Stack

| Layer      | Technology               | Why                                                       |
|------------|--------------------------|-----------------------------------------------------------|
| Backend    | Go 1.24 + Gin            | Low latency, small memory footprint, excellent concurrency |
| AI         | Eino (CloudWeGo)         | Production-ready Go LLM orchestration, graph-based agents |
| Database   | PostgreSQL 16            | ACID, JSONB, full-text, partitioning, future sharding     |
| Cache      | Redis 7                  | Short-term memory, session state, OAuth CSRF tokens       |
| VectorDB   | Qdrant                   | High-perf ANN search, payload filtering, gRPC API         |
| Storage    | MinIO                    | S3-compatible, on-prem, presigned URLs                    |
| Auth       | JWT + Google OAuth       | Stateless access tokens, rotating refresh tokens          |
| Frontend   | React 18 + Vite + TS     | Fast HMR, type-safe, Tailwind for UI                      |
| Deploy     | Docker Compose → K8s     | Start simple, extract when needed                         |

---

## Architecture

### Modular Monolith with Clean Architecture

```
backend/
├── cmd/server/            # Entrypoint, DI wiring
└── internal/
    ├── shared/            # Cross-cutting: config, db, cache, eventbus, middleware
    └── modules/
        ├── auth/          # JWT, Google OAuth, RBAC
        ├── user/          # Profile, preferences
        ├── chat/          # Conversations, messages, streaming
        ├── agent/         # Agent definitions, Eino orchestration
        ├── memory/        # Short/long/semantic memory
        ├── rag/           # Chunking, embedding, retrieval, reranking
        ├── document/      # Upload, MinIO, processing pipeline
        ├── workflow/      # DAG engine, triggers, runs
        ├── tool/          # Tool registry, pluggable executors
        └── voice/         # Audio upload, Whisper transcription
```

Each module follows **Clean Architecture**:
```
module/
├── domain/
│   ├── entity/     # Aggregates, entities, value objects (no imports from infra)
│   └── event/      # Domain events published to the internal event bus
├── application/
│   ├── dto/        # Request/response data transfer objects
│   ├── port/       # Repository & service interfaces (contracts)
│   └── service/    # Business logic (depends only on ports)
├── infrastructure/
│   ├── repository/ # pgx implementations of port interfaces
│   └── ...         # OAuth clients, MinIO adapters, Qdrant clients
└── delivery/
    └── http/       # Gin handlers and route registration
```

**Why this layout?**
- `domain` has zero external imports → portable across any infra change
- `application/service` is 100% testable without a database
- `delivery` is swappable: add gRPC, WebSocket, or CLI delivery alongside HTTP
- Future microservice extraction = move one module folder + its DB tables

---

## RAG Architecture

```
Upload Document
  → MinIO (raw file storage)
  → Processing pipeline (background worker)
  → Chunking (fixed-size + semantic sentence boundaries)
  → Embedding (OpenAI text-embedding-3-small, 1536 dims)
  → Qdrant (vector upsert with user_id payload filter)
  → PostgreSQL (chunk metadata + document status update)
  → Event: EmbeddingCreated → audit log

Query Time
  User Query
  → Embedding (same model)
  → Qdrant ANN search (filtered by user_id, top-k=20)
  → PostgreSQL join (enrich with doc metadata)
  → Cross-encoder reranking (top-5)
  → Context builder (token-budget aware)
  → Eino Agent (injected as system context)
```

---

## Memory Architecture

| Layer          | Storage | TTL / Eviction                        |
|----------------|---------|---------------------------------------|
| Short-term     | Redis   | 2-hour sliding window (last N turns)  |
| Long-term      | PostgreSQL | Permanent, indexed by importance   |
| Semantic       | Qdrant  | Mirrors long-term, vector searchable  |

Memory entries are classified: FACT, PREFERENCE, EVENT, SKILL, RELATIONSHIP.
The agent retrieves top-K relevant memories before each response.

---

## AI Agent Architecture (Eino)

```
User Message
  │
  ▼
Supervisor Agent (Eino Graph)
  ├── Research Agent    → RAG retrieval + web search tools
  ├── Coding Agent      → GitHub tool + code execution
  ├── Planning Agent    → workflow creation, task breakdown
  └── Workflow Agent    → triggers existing workflows

Each agent:
  1. Loads short-term memory from Redis
  2. Retrieves semantic memories from Qdrant
  3. Retrieves RAG context if rag_enabled
  4. Calls tools via the Tool Registry
  5. Writes new memories post-response
  6. Emits ChatCompleted event
```

---

## RBAC

| Role          | Permissions                                              |
|---------------|----------------------------------------------------------|
| `USER`        | Own resources: chat, documents, memory, agents, workflows |
| `PREMIUM_USER`| USER + higher rate limits, voice, advanced agents        |
| `ADMIN`       | All resources, user management, audit logs               |

Role is embedded in the JWT claim — no DB lookup per request.

---

## Internal Event Bus

The in-process event bus decouples modules without a message broker. It will be
extracted to NATS or Kafka when the first module is split into a microservice.

| Event                  | Publisher | Subscribers              |
|------------------------|-----------|--------------------------|
| `auth.user_registered` | auth      | audit, billing, email    |
| `auth.user_logged_in`  | auth      | audit                    |
| `document.uploaded`    | document  | rag (triggers processing)|
| `rag.embedding_created`| rag       | audit                    |
| `memory.created`       | memory    | audit                    |
| `workflow.executed`    | workflow  | audit, billing           |
| `chat.completed`       | chat      | memory (auto-extraction) |

---

## Database Schema

13 tables across 11 migration files (run in order):

```
000_init.sql              Extensions, shared trigger function
001_create_users.sql      Users with RBAC + OAuth support
002_create_refresh_tokens.sql  JWT rotation, stored hashed
003_create_agents.sql     Agent definitions
004_create_conversations.sql   Chat session containers
005_create_messages.sql   Individual chat messages
006_create_documents.sql  Uploaded document metadata
007_create_document_chunks.sql Chunked text + Qdrant links
008_create_memories.sql   Long-term memory entries
009_create_workflows.sql  Workflow DAGs + execution runs
010_create_tools.sql      Tool registry + user configs
011_create_audit_logs.sql Partitioned, immutable audit trail
```

---

## Development Roadmap

| Phase | Feature         | Modules            | Target       |
|-------|-----------------|--------------------|--------------|
| 1     | Authentication  | auth               | Week 1       |
| 2     | Chat            | chat, agent (basic)| Week 2-3     |
| 3     | RAG             | document, rag      | Week 4-5     |
| 4     | Memory          | memory             | Week 6       |
| 5     | AI Agents       | agent (Eino graph) | Week 7-9     |
| 6     | Voice           | voice              | Week 10      |
| 7     | Workflows       | workflow, tool     | Week 11-13   |
| 8     | Multi-tenant    | all modules        | Week 14-16   |

---

## Quick Start

```bash
# 1. Clone and configure
git clone <repo>
cd jarvas
cp .env.example .env
# Fill in JWT_SECRET, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, OPENAI_API_KEY

# 2. Start everything (first time)
make setup

# 3. Dev mode (two terminals)
make dev          # Go backend on :8080
make dev-front    # React frontend on :5173

# 4. Or run fully in Docker
make docker-up
```

---

## API Overview

All endpoints are under `/api/v1`.

### Auth
```
POST   /auth/register          Email/password registration
POST   /auth/login             Email/password login → TokenPair
POST   /auth/refresh           Rotate refresh token (cookie or body)
POST   /auth/logout            Revoke refresh token
GET    /auth/google/login      Get Google OAuth URL
GET    /auth/google/callback   Handle OAuth callback
GET    /auth/me                Get current user (protected)
```

### Chat (Phase 2)
```
POST   /conversations          Create conversation
GET    /conversations          List conversations
GET    /conversations/:id      Get conversation + messages
POST   /conversations/:id/messages   Send message (stream=true for SSE)
DELETE /conversations/:id      Archive conversation
```

### Documents (Phase 3)
```
POST   /documents              Upload document (multipart)
GET    /documents              List documents
GET    /documents/:id          Get document metadata
DELETE /documents/:id          Delete document + chunks
GET    /documents/:id/chunks   List chunks
POST   /rag/search             Semantic search
```

### Memory (Phase 4)
```
GET    /memories               List memories
POST   /memories               Create memory
PATCH  /memories/:id           Update memory
DELETE /memories/:id           Delete memory
POST   /memories/search        Semantic memory search
```

### Agents (Phase 5)
```
POST   /agents                 Create agent
GET    /agents                 List agents
GET    /agents/:id             Get agent
PATCH  /agents/:id             Update agent
DELETE /agents/:id             Delete agent
```

### Workflows (Phase 7)
```
POST   /workflows              Create workflow
GET    /workflows              List workflows
POST   /workflows/:id/run      Trigger workflow run
GET    /workflows/:id/runs     List runs
```

---

## Security Considerations

- **JWT**: HS256 signed, 15min access token, 7-day rotating refresh token
- **Refresh tokens**: stored as SHA-256 hash only — token theft doesn't expose raw value
- **OAuth state**: stored in Redis with 10min TTL to prevent CSRF
- **Passwords**: bcrypt with cost 10
- **Rate limiting**: per-IP and per-user (Redis sliding window — Phase 2)
- **Audit log**: immutable, partitioned by quarter, captures all sensitive actions
- **Tool credentials**: encrypted at application layer before DB storage (Phase 5)
- **CORS**: strict allowlist, credentials allowed only for configured origins

---

## Future Microservice Extraction Strategy

The boundary is already drawn by the module structure. Extraction order:

1. `document` + `rag` → Document Processing Service (I/O heavy, scales independently)
2. `voice` → Voice Service (GPU/Whisper, latency-sensitive)
3. `workflow` → Workflow Engine (stateful, long-running)
4. `billing` → Billing Service (isolated domain, compliance boundary)
5. `agent` → Agent Service (compute-heavy, independent scaling)

Each extraction requires:
- Moving the module folder
- Replacing the internal event bus calls with NATS/Kafka publish
- Replacing direct function calls with gRPC or HTTP

The repository pattern means DB access is already abstracted behind interfaces.
