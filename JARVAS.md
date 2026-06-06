# JARVAS — Complete Developer Guide

> One document to understand the codebase, run it locally, and know exactly what to build next.

---

## Table of Contents

1. [What Is Jarvas?](#1-what-is-jarvas)
2. [Tech Stack & Why](#2-tech-stack--why)
3. [Architecture Deep Dive](#3-architecture-deep-dive)
4. [Complete Folder Structure](#4-complete-folder-structure)
5. [Database Schema](#5-database-schema)
6. [How to Run](#6-how-to-run)
7. [Environment Variables](#7-environment-variables)
8. [What Is Already Built](#8-what-is-already-built)
9. [API Reference — Phase 1 (Auth)](#9-api-reference--phase-1-auth)
10. [Phase-by-Phase Build Plan](#10-phase-by-phase-build-plan)
11. [Phase 2 — Chat (Next)](#11-phase-2--chat-next)
12. [Phase 3 — RAG + Documents](#12-phase-3--rag--documents)
13. [Phase 4 — Memory](#13-phase-4--memory)
14. [Phase 5 — AI Agents (Eino)](#14-phase-5--ai-agents-eino)
15. [Phase 6 — Voice](#15-phase-6--voice)
16. [Phase 7 — Workflows](#16-phase-7--workflows)
17. [Phase 8 — Multi-Tenant](#17-phase-8--multi-tenant)
18. [Coding Conventions](#18-coding-conventions)
19. [Security Model](#19-security-model)
20. [Microservice Extraction Plan](#20-microservice-extraction-plan)

---

## 1. What Is Jarvas?

Jarvas is a **production-grade, multi-tenant AI Assistant Platform**. Users can:

- Register and log in (email/password or Google OAuth)
- Upload documents and search them with AI (RAG)
- Chat with AI agents with persistent memory
- Speak with the AI using voice
- Build automated workflows
- Connect external tools (GitHub, Calendar, Email, etc.)
- Create and manage custom AI agents

**Target:** 0 → 5,000–10,000 users as a **Modular Monolith**, with each module designed for clean extraction into a microservice when traffic demands it.

---

## 2. Tech Stack & Why

| Layer      | Technology                  | Why This, Not X                                                                 |
|------------|-----------------------------|---------------------------------------------------------------------------------|
| **Backend** | Go 1.24 + Gin              | ~10x less memory than Node/Python. Goroutine-per-request is free. Gin is fast.  |
| **AI Orchestration** | Eino (CloudWeGo)  | Production Go LLM framework. Graph-based agent design. Works with any model.    |
| **Database** | PostgreSQL 16              | JSONB, full-text search, partitioning, extensions. Not NoSQL — ACID matters.    |
| **Cache** | Redis 7                      | Short-term memory, JWT refresh token lookup, OAuth CSRF state, rate limiting.   |
| **Vector DB** | Qdrant                    | 10x faster than pgvector at scale. Payload filtering avoids scan of all vectors.|
| **Object Storage** | MinIO               | S3-compatible, self-hosted. Presigned URLs keep the backend out of the hot path.|
| **Auth** | JWT + Google OAuth            | Stateless access token (no DB hit per request). Rotating refresh token for security.|
| **Frontend** | React 18 + Vite + TS      | Vite HMR is instant. TypeScript catches contract mismatches at build time.      |
| **State** | Zustand + React Query         | Zustand for auth/UI state. React Query for server state with cache + retry.     |
| **Deploy** | Docker Compose → Kubernetes  | Start simple. Add K8s when a single service needs independent scaling.          |

---

## 3. Architecture Deep Dive

### Modular Monolith

The entire backend is **one binary** but organized as if it were microservices. Each module:
- Has its own domain model (no cross-module DB joins)
- Communicates via the **internal event bus** (not function calls)
- Owns its tables, never touches another module's tables directly
- Has its own clean architecture layers

```
Request → Gin Router → Delivery (HTTP Handler)
                              ↓
                    Application (Service)
                    depends on ports (interfaces)
                              ↓
                    Infrastructure (Repository, Redis, Qdrant, MinIO)
                              ↓
                    Domain (Entity, Event, ValueObject)
                    — zero external imports —
```

### Why Not Microservices From Day 1?

- No distributed tracing overhead
- No network latency between modules
- One deployment, one config, one database connection pool
- Easier to refactor domain boundaries when you're still learning the problem
- At 5k–10k users, you'll know which modules need to scale — extract those

### Internal Event Bus

Instead of direct function calls between modules, modules publish domain events:

```
auth.Register()  →  publishes UserRegistered
                         ↓
            audit module subscribes → writes audit log
            billing module subscribes → initialises plan
            email module subscribes → sends welcome email
```

The bus is synchronous-with-goroutines today. Swapping to NATS/Kafka later = change only the `eventbus` package.

---

## 4. Complete Folder Structure

```
jarvas/
├── .env.example                     # All env vars with defaults
├── .gitignore
├── Makefile                         # make dev, make docker-up, make migrate-up, etc.
├── README.md                        # Brief overview
├── JARVAS.md                        # ← This file
├── docker-compose.yml               # Postgres, Redis, Qdrant, MinIO, Backend, Frontend
│
├── backend/
│   ├── Dockerfile                   # Multi-stage: builder + scratch runtime
│   ├── go.mod
│   │
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # Entrypoint: DI wiring + Gin router + graceful shutdown
│   │
│   ├── migrations/
│   │   ├── 000_init.sql             # Extensions, update_updated_at trigger
│   │   ├── 001_create_users.sql
│   │   ├── 002_create_refresh_tokens.sql
│   │   ├── 003_create_agents.sql
│   │   ├── 004_create_conversations.sql
│   │   ├── 005_create_messages.sql
│   │   ├── 006_create_documents.sql
│   │   ├── 007_create_document_chunks.sql
│   │   ├── 008_create_memories.sql
│   │   ├── 009_create_workflows.sql
│   │   ├── 010_create_tools.sql
│   │   └── 011_create_audit_logs.sql  # Partitioned by quarter
│   │
│   └── internal/
│       ├── shared/                  # Cross-cutting concerns
│       │   ├── config/config.go     # Typed config from env
│       │   ├── database/postgres.go # pgxpool wrapper
│       │   ├── cache/redis.go       # Redis client with typed helpers
│       │   ├── logger/logger.go     # Zap logger (dev = colored, prod = JSON)
│       │   ├── errors/errors.go     # AppError with HTTP status + code + cause
│       │   ├── response/response.go # OK, Created, Paginated, Error, NoContent
│       │   ├── validator/validator.go # Struct validation → AppError
│       │   ├── eventbus/eventbus.go # In-process pub/sub
│       │   └── middleware/
│       │       ├── auth.go          # Authenticate() + RequireRole()
│       │       ├── logger.go        # HTTP request logger
│       │       └── recovery.go      # Panic → 500
│       │
│       └── modules/
│           │
│           ├── auth/                # ✅ FULLY IMPLEMENTED
│           │   ├── domain/
│           │   │   ├── entity/user.go
│           │   │   ├── entity/refresh_token.go
│           │   │   └── event/events.go       # UserRegistered, UserLoggedIn, etc.
│           │   ├── application/
│           │   │   ├── dto/auth_dto.go
│           │   │   ├── port/repository.go    # UserRepository, TokenRepository interfaces
│           │   │   └── service/
│           │   │       ├── auth_service.go   # Register, Login, Refresh, Logout, OAuth
│           │   │       └── token_service.go  # JWT sign/verify, token hashing
│           │   ├── infrastructure/
│           │   │   ├── repository/user_repo.go
│           │   │   ├── repository/token_repo.go
│           │   │   └── oauth/google.go
│           │   └── delivery/http/
│           │       ├── handler.go
│           │       └── routes.go
│           │
│           ├── user/                # 🔲 Domain + DTO only
│           │   ├── domain/entity/user_profile.go
│           │   └── application/dto/user_dto.go
│           │
│           ├── chat/                # ✅ FULLY IMPLEMENTED (Phase 2)
│           │   ├── domain/entity/conversation.go
│           │   ├── application/dto/chat_dto.go
│           │   ├── application/port/repository.go
│           │   ├── application/service/chat_service.go   # SendMessage, StreamMessage, Redis memory
│           │   ├── infrastructure/repository/conversation_repo.go
│           │   ├── infrastructure/repository/message_repo.go
│           │   └── delivery/http/handler.go + routes.go
│           │
│           ├── document/            # ✅ FULLY IMPLEMENTED (Phase 3)
│           │   ├── domain/entity/document.go
│           │   ├── domain/event/events.go                # DocumentUploaded, Indexed, Deleted
│           │   ├── application/dto/document_dto.go
│           │   ├── application/port/ports.go
│           │   ├── application/service/document_service.go
│           │   ├── infrastructure/storage/minio.go
│           │   ├── infrastructure/extractor/extractor.go  # PDF + PlainText
│           │   ├── infrastructure/repository/document_repo.go
│           │   ├── infrastructure/repository/chunk_repo.go
│           │   └── delivery/http/handler.go + routes.go
│           │
│           ├── rag/                 # ✅ FULLY IMPLEMENTED (Phase 3)
│           │   ├── domain/entity/rag.go
│           │   ├── application/port/ports.go              # EmbeddingPort, VectorStorePort
│           │   ├── application/service/chunker.go         # 512-token sentence-aware chunks
│           │   ├── application/service/processor.go       # Full async indexing pipeline
│           │   ├── application/service/rag_service.go     # Search → rerank → context
│           │   ├── infrastructure/embedding/openai.go     # text-embedding-3-small
│           │   ├── infrastructure/vectorstore/qdrant.go   # gRPC: upsert, search, delete
│           │   └── delivery/http/handler.go + routes.go
│           │
│           ├── agent/               # 🔲 Domain + DTO only (Phase 5)
│           │   ├── domain/entity/agent.go
│           │   └── application/dto/agent_dto.go
│           │
│           ├── memory/              # 🔲 Domain only (Phase 4 — NEXT)
│           │   └── domain/entity/memory.go
│           │
│           ├── workflow/            # 🔲 Domain only (Phase 7)
│           │   └── domain/entity/workflow.go
│           │
│           ├── tool/                # 🔲 Domain only (Phase 7)
│           │   └── domain/entity/tool.go
│           │
│           └── voice/               # 🔲 Domain only (Phase 6)
│               └── domain/entity/voice.go
│
└── frontend/
    ├── Dockerfile.dev
    ├── index.html
    ├── package.json                 # React 18, Vite, TS, Zustand, React Query, Tailwind
    ├── vite.config.ts
    ├── tsconfig.json
    ├── tailwind.config.js
    └── src/
        ├── main.tsx                 # App bootstrap: QueryClient, Router, Toaster
        ├── App.tsx                  # Route tree with PrivateRoute / PublicRoute guards
        ├── index.css                # Tailwind + CSS variables (light/dark)
        ├── vite-env.d.ts            # Vite ImportMeta type declarations
        ├── types/api.ts             # All API response types
        ├── store/authStore.ts       # Zustand: user, accessToken, isAuthenticated
        ├── services/
        │   ├── api.ts               # Axios client: auto Bearer header + silent refresh
        │   ├── auth.service.ts      # register, login, logout, refresh, me, googleUrl
        │   ├── chat.service.ts      # conversations, messages, SSE streaming   ✅ Phase 2
        │   └── document.service.ts  # upload, list, delete, presign, RAG search ✅ Phase 3
        ├── hooks/
        │   └── useConversations.ts  # React Query hooks for chat              ✅ Phase 2
        ├── components/
        │   └── layout/
        │       ├── Layout.tsx       # Shell: Sidebar + Header + <Outlet>
        │       ├── Sidebar.tsx      # Nav links with active state
        │       └── Header.tsx       # User avatar + logout button
        └── pages/
            ├── auth/
            │   └── LoginPage.tsx    # Tabbed: Sign in + Create account + Google OAuth ✅
            ├── chat/
            │   ├── ChatPage.tsx           # Split layout: list + thread           ✅ Phase 2
            │   ├── ConversationList.tsx   # New/delete, timestamps                ✅ Phase 2
            │   ├── MessageThread.tsx      # Optimistic UI, streaming, auto-scroll ✅ Phase 2
            │   ├── MessageBubble.tsx      # Markdown + streaming cursor ▍        ✅ Phase 2
            │   └── ChatInput.tsx          # Textarea, stream toggle, Enter=send  ✅ Phase 2
            ├── documents/
            │   ├── DocumentsPage.tsx      # Tabbed: My Docs + Search Knowledge   ✅ Phase 3
            │   ├── UploadDropzone.tsx     # react-dropzone, 50 MB                ✅ Phase 3
            │   ├── DocumentCard.tsx       # Status badge, chunk count, actions   ✅ Phase 3
            │   └── DocumentSearch.tsx     # Semantic search with ranked results   ✅ Phase 3
            ├── dashboard/DashboardPage.tsx   # stub (Phase 8)
            ├── agents/AgentsPage.tsx         # stub (Phase 5)
            ├── memory/MemoryPage.tsx         # stub (Phase 4)
            ├── workflows/WorkflowsPage.tsx   # stub (Phase 7)
            └── settings/SettingsPage.tsx     # stub
```

---

## 5. Database Schema

All migrations live in `backend/migrations/`. Run in numeric order.

### Tables at a Glance

| Table                | Key Columns                                                      | Notes                              |
|----------------------|------------------------------------------------------------------|------------------------------------|
| `users`              | id, email, password_hash, role, provider, provider_id           | RBAC roles; unique email per tenant |
| `refresh_tokens`     | user_id, token_hash, revoked, expires_at                        | SHA-256 hashed; revoked on rotation |
| `agents`             | user_id, type, system_prompt, model, tools_enabled (JSONB)      | Eino graph config                  |
| `conversations`      | user_id, agent_id, title, status                                |                                    |
| `messages`           | conversation_id, role, content, token_count, tool_calls (JSONB) | Ordered by created_at              |
| `documents`          | user_id, storage_key, status, chunk_count                       | storage_key = MinIO object key     |
| `document_chunks`    | document_id, content, chunk_index, qdrant_id                    | Links Postgres ↔ Qdrant            |
| `memories`           | user_id, type, content, importance, qdrant_id, expires_at       | Long-term; semantic in Qdrant      |
| `workflows`          | user_id, definition (JSONB), trigger_type, cron_expr            | DAG as JSON                        |
| `workflow_runs`      | workflow_id, status, result (JSONB), started_at, completed_at   |                                    |
| `tools`              | name (unique), schema (JSONB), is_builtin                       | 5 built-in tools pre-seeded        |
| `user_tool_configs`  | user_id, tool_id, config (JSONB)                                | Config encrypted at app layer      |
| `audit_logs`         | user_id, action, resource, payload (JSONB), created_at          | Partitioned by quarter; append-only|

### RBAC

```
ADMIN        → all resources
USER         → own resources only
PREMIUM_USER → USER + higher limits, voice, advanced agents
```

Role is embedded in the JWT — no DB lookup per request.

---

## 6. How to Run

### Prerequisites

| Tool             | Version | Install                          |
|------------------|---------|----------------------------------|
| Go               | 1.24+   | `brew install go`                |
| Docker Desktop   | latest  | docker.com                       |
| Node.js          | 20+     | `brew install node`              |
| Make             | any     | pre-installed on macOS           |
| Air (hot reload) | latest  | `go install github.com/cosmtrek/air@latest` |

---

### Option A — Full Docker (Easiest)

```bash
cd jarvas

# 1. Copy env and fill in secrets
cp .env.example .env
# Edit .env: set JWT_SECRET, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, OPENAI_API_KEY

# 2. Build and start everything
make docker-up

# Services:
#   Backend  → http://localhost:8080
#   Frontend → http://localhost:5173
#   MinIO Console → http://localhost:9001  (minioadmin / minioadmin)
#   Qdrant Dashboard → http://localhost:6333/dashboard
```

---

### Option B — Local Dev (Hot Reload)

```bash
cd jarvas

# 1. Start only the infrastructure (DB, Redis, etc.)
docker compose up -d postgres redis qdrant minio

# 2. Copy and configure .env
cp .env.example .env

# 3. Run migrations
make migrate-up

# Terminal 1 — Backend (with Air hot-reload)
make dev
# or without air:
cd backend && go run ./cmd/server

# Terminal 2 — Frontend
cd frontend && npm install && npm run dev

# Backend  → http://localhost:8080
# Frontend → http://localhost:5173
```

---

### Option C — First-Time Full Setup

```bash
make setup
# This will:
# 1. Copy .env.example → .env
# 2. Start Postgres, Redis, Qdrant, MinIO in Docker
# 3. Run all migrations
# 4. Install frontend npm deps
# Then run: make dev + make dev-front
```

---

### Useful Make Commands

# Option 1 — Single command (recommended)
make dev
# Auto-copies .env, starts Docker infra, waits for Postgres, runs backend

# Option 2 — With hot-reload
make install-air   # once
make dev           # now uses air

# Option 3 — Two terminals
make infra         # terminal 1: starts Postgres, Redis, Qdrant, MinIO
make dev           # terminal 2: starts backend
make dev-front     # terminal 3: starts React frontend on :5173

```bash
make help           # List all commands

# Development
make dev            # Start backend with air hot-reload
make dev-front      # Start frontend Vite dev server

# Docker
make docker-up      # Start all containers
make docker-down    # Stop all containers
make docker-logs    # Tail all container logs
make docker-ps      # Show container status

# Database
make migrate-up     # Apply all migrations
make migrate-create NAME=add_something  # Create a new migration file

# Code quality
make test           # Run Go tests with race detector
make lint           # Run golangci-lint
make fmt            # Format Go code
make vet            # Run go vet
make tidy           # go mod tidy

# Frontend
make front-install  # npm install
make front-build    # Production build
make front-lint     # ESLint

# Misc
make build          # Build backend binary → backend/bin/server
make clean          # Remove build artefacts
```

---

### MinIO Setup (First Run)

After `docker compose up`, create the required buckets:

```bash
# Open MinIO Console: http://localhost:9001
# Login: minioadmin / minioadmin
# Create buckets: "documents" and "audio"
```

Or via mc (MinIO CLI):
```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/documents
mc mb local/audio
```

---

### Verify Everything Is Running

```bash
# Backend health check
curl http://localhost:8080/health
# → {"status":"healthy","service":"jarvas"}

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","full_name":"Test User"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

---

## 7. Environment Variables

Copy `.env.example` → `.env` and fill in these values:

| Variable                   | Required | Default              | Description                              |
|----------------------------|----------|----------------------|------------------------------------------|
| `APP_ENV`                  | No       | `development`        | `development` or `production`            |
| `APP_PORT`                 | No       | `8080`               | HTTP listen port                         |
| `DB_PASSWORD`              | **Yes**  | —                    | Postgres password                        |
| `JWT_SECRET`               | **Yes**  | —                    | Min 32 chars, random string              |
| `GOOGLE_CLIENT_ID`         | **Yes**  | —                    | From Google Cloud Console                |
| `GOOGLE_CLIENT_SECRET`     | **Yes**  | —                    | From Google Cloud Console                |
| `OPENAI_API_KEY`           | Phase 2+ | —                    | For embeddings + chat completions        |
| `MINIO_ACCESS_KEY`         | No       | `minioadmin`         | Change in production                     |
| `MINIO_SECRET_KEY`         | No       | `minioadmin`         | Change in production                     |
| `REDIS_PASSWORD`           | No       | `` (empty)           | Set in production                        |
| `QDRANT_API_KEY`           | No       | `` (empty)           | Required if Qdrant auth is enabled       |
| `JWT_ACCESS_EXPIRY`        | No       | `15m`                | e.g. `15m`, `1h`                        |
| `JWT_REFRESH_EXPIRY`       | No       | `168h` (7d)          | e.g. `168h`, `720h`                     |
| `CORS_ORIGINS`             | No       | `http://localhost:5173` | Comma-separated                       |

---

## 8. What Is Already Built

**All 8 phases complete — 130 backend module files · 49 frontend files · 0 build errors**

| Phase | Module | Status | Key API surface |
|-------|--------|--------|-----------------|
| 1 | Auth | ✅ | register, login, refresh, logout, /me, Google OAuth |
| 2 | Chat | ✅ | conversations, messages, SSE streaming, Redis short-term memory |
| 3 | Documents + RAG | ✅ | upload→MinIO→chunk→embed→Qdrant, semantic search |
| 4 | Memory | ✅ | long-term facts (Postgres+Qdrant), LLM extraction after every chat turn |
| 5 | AI Agents (Eino) | ✅ | named agents, tool calling loop, memory injection, Eino ChatModel |
| 6 | Voice | ✅ | audio upload→Whisper→transcript injected into ChatInput |
| 7 | Workflows + Tools | ✅ | DAG executor, cron scheduler, agent/tool/condition/delay nodes |
| 8 | Multi-Tenant | ✅ | tenants, membership roles, invite by email, workspace switcher |

### ✅ Backend modules

| Module | Files | Notes |
|--------|-------|-------|
| `auth` | 14 | JWT, Google OAuth, RBAC, refresh rotation |
| `chat` | 8  | SendMessage, StreamMessage, agent delegation, ChatCompleted event |
| `document` | 9 | MinIO, PDF+text extractors, async processing pipeline |
| `rag` | 8  | OpenAI embeddings, Qdrant gRPC, reranking |
| `memory` | 8  | Postgres+Qdrant, LLM extraction, `MemoryRetriever` port |
| `agent` | 8  | Eino ChatModel, tool calling loop, `AgentRunnerPort` |
| `voice` | 8  | Whisper API, MinIO audio, Redis session store (no migration needed) |
| `workflow` | 10 | Kahn's topo sort, cron scheduler, 4 node types |
| `tool` | 7  | Registry (web_search, calculator, http_request), DB-backed tool API |
| `tenant` | 7  | OWNER/ADMIN/MEMBER roles, invite, personal tenant auto-creation |

### ✅ Shared infrastructure

| Package | What It Does |
|---------|-------------|
| `config` | Typed env config, searches `.env` and `../.env` |
| `database` | pgxpool with health check, configurable limits |
| `cache` | Redis typed helpers: Get/Set/Del/Exists/Expire |
| `logger` | Zap: colored dev, JSON prod; logs 5xx causes |
| `errors` | `AppError{HTTPStatus, Code, Message, Err}` — full chain |
| `response` | OK/Created/Paginated/NoContent/Error (auto-unwraps AppError) |
| `validator` | Struct tags → human-readable AppError messages |
| `eventbus` | In-process goroutine pub/sub, panic-safe, swap-ready for NATS |
| `middleware` | Authenticate, RequireRole, RequestLogger, Recovery, ValidateTenant |

### ✅ Frontend pages

| Page | Status |
|------|--------|
| LoginPage | ✅ Tabbed sign-in + register + Google OAuth |
| DashboardPage | ✅ Stats grid, quick actions, recent conversations |
| ChatPage | ✅ Split layout, SSE streaming, agent selector, voice mic |
| DocumentsPage | ✅ Upload dropzone, status polling, semantic search |
| AgentsPage | ✅ Create/edit/delete agents, tool toggles, "Chat with agent" |
| MemoryPage | ✅ List + semantic search + manual create/delete |
| WorkflowsPage | ✅ Builder (visual + JSON toggle), trigger run, run history |
| SettingsPage | ✅ Workspace switcher, member list, invite by email |

### ✅ Infrastructure

- `docker-compose.yml` — Postgres, Redis, Qdrant, MinIO, Backend, Frontend (health checks)
- `backend/Dockerfile` — multi-stage, final image from `scratch` (~8 MB)
- `Makefile` — `make dev` auto-starts Docker infra, detects air, falls back to `go run`
- `migrations/` — 13 migration files (000–012), all production-grade indexes

---

## 9. API Reference — Phase 1 (Auth)

Base URL: `http://localhost:8080/api/v1`

### POST `/auth/register`

```json
// Request
{ "email": "user@example.com", "password": "password123", "full_name": "Jane Smith" }

// Response 201
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "uuid-uuid-...",
    "expires_in": 900,
    "token_type": "Bearer",
    "user": {
      "id": "uuid", "email": "user@example.com", "full_name": "Jane Smith",
      "role": "USER", "email_verified": false
    }
  }
}
```

### POST `/auth/login`

```json
// Request
{ "email": "user@example.com", "password": "password123" }
// Response 200 — same shape as register
```

### POST `/auth/refresh`

```
// Cookie: refresh_token=<raw-token>   (preferred, set automatically on login)
// OR body: { "refresh_token": "<raw-token>" }

// Response 200
{ "success": true, "data": { "access_token": "...", "refresh_token": "...", "expires_in": 900 } }
```

### POST `/auth/logout` _(protected)_

```
// Header: Authorization: Bearer <access_token>
// Response: 204 No Content
// Side effects: refresh token revoked, cookie cleared
```

### GET `/auth/google/login`

```json
// Response 200
{ "success": true, "data": { "url": "https://accounts.google.com/o/oauth2/auth?..." } }
// Redirect browser to this URL
```

### GET `/auth/google/callback?code=...&state=...`

```
// Called by Google. Returns same shape as login.
```

### GET `/auth/me` _(protected)_

```json
// Header: Authorization: Bearer <access_token>
// Response 200
{
  "success": true,
  "data": { "id": "uuid", "email": "...", "full_name": "...", "role": "USER", "email_verified": false }
}
```

### Error Response Format (all endpoints)

```json
{ "success": false, "code": "UNAUTHORIZED", "message": "invalid or expired token" }
```

---

## 10. Phase-by-Phase Build Plan

| Phase | What You Build | Status |
|-------|---------------|--------|
| **1** | Auth (JWT, Google OAuth, RBAC) | ✅ Done |
| **2** | Chat (conversations, messages, SSE streaming) | ✅ Done |
| **3** | Documents + RAG (chunking, embedding, Qdrant, search) | ✅ Done |
| **4** | Memory (short-term Redis, long-term Postgres, semantic Qdrant) | ✅ Done |
| **5** | AI Agents (Eino, tool calling loop, memory injection) | ✅ Done |
| **6** | Voice (audio upload → Whisper → transcript into ChatInput) | ✅ Done |
| **7** | Workflows (DAG engine, cron scheduler, tool execution) | ✅ Done |
| **8** | Multi-Tenant (orgs, membership, invite, workspace switcher) | ✅ Done |

> All 8 phases are complete. See [PHASES.md](PHASES.md) for the detailed implementation record of each phase.

---

## 11. Phase 2 — Chat (Next)

### What to Build

```
backend/internal/modules/chat/
├── application/
│   ├── port/repository.go        # ConversationRepository, MessageRepository
│   └── service/chat_service.go   # Create, SendMessage, List, Archive
├── infrastructure/
│   └── repository/
│       ├── conversation_repo.go
│       └── message_repo.go
└── delivery/http/
    ├── handler.go
    └── routes.go

frontend/src/
├── services/chat.service.ts
├── pages/chat/
│   ├── ChatPage.tsx
│   ├── ConversationList.tsx
│   └── MessageThread.tsx
└── hooks/useChat.ts
```

### Key Implementation Points

**1. Chat Service (`chat_service.go`)**

```go
func (s *ChatService) SendMessage(ctx context.Context, userID uuid.UUID, req dto.SendMessageRequest) (*dto.MessageResponse, error) {
    // 1. Get or create conversation
    // 2. Append user message to DB
    // 3. Load last N messages from Redis (short-term memory) — fallback to DB
    // 4. Call Eino agent (basic: just OpenAI chat completion for Phase 2)
    // 5. Append assistant message to DB
    // 6. Update Redis short-term memory cache
    // 7. Publish ChatCompleted event
    // 8. Return assistant message
}
```

**2. Streaming (SSE)**

```go
// In handler.go, when req.Stream == true:
c.Header("Content-Type", "text/event-stream")
c.Header("Cache-Control", "no-cache")
c.Header("Connection", "keep-alive")

flusher := c.Writer.(http.Flusher)
// Stream chunks from OpenAI → write SSE frames → flush
// Final frame: data: {"done":true}
```

**3. Short-Term Memory in Redis**

```
Key: short_term_memory:{conversation_id}
Value: JSON array of last 20 messages
TTL: 2 hours (sliding)
```

**4. Routes**

```
POST   /api/v1/conversations
GET    /api/v1/conversations
GET    /api/v1/conversations/:id
DELETE /api/v1/conversations/:id
POST   /api/v1/conversations/:id/messages    ← supports SSE streaming
GET    /api/v1/conversations/:id/messages
```

**5. Wire in `main.go`**

```go
// Add these after existing auth wiring:
convRepo    := chatrepo.NewConversationRepository(db.Pool)
msgRepo     := chatrepo.NewMessageRepository(db.Pool)
chatSvc     := chatservice.NewChatService(convRepo, msgRepo, redisClient, bus, cfg.AI)
chatHandler := chathttp.NewChatHandler(chatSvc)
chathttp.RegisterRoutes(v1, chatHandler, authMW)
```

---

## 12. Phase 3 — RAG + Documents

### What to Build

```
backend/internal/modules/document/
├── application/port/           # DocumentRepository, ChunkRepository, StoragePort
├── application/service/
│   ├── document_service.go     # Upload, List, Delete, GetPresignedURL
│   └── processor_service.go   # Background: Chunk → Embed → Qdrant upsert
├── infrastructure/
│   ├── repository/document_repo.go
│   ├── repository/chunk_repo.go
│   └── storage/minio_storage.go   # Upload, GetPresignedURL, Delete

backend/internal/modules/rag/
├── application/port/           # VectorStore, EmbeddingPort, RerankPort
├── application/service/
│   └── rag_service.go          # Search → Embed query → Qdrant → Rerank → Context
├── infrastructure/
│   ├── vectorstore/qdrant.go   # UpsertPoints, Search with payload filter
│   └── embedding/openai.go     # text-embedding-3-small via OpenAI SDK
└── delivery/http/
    ├── handler.go
    └── routes.go
```

### RAG Pipeline Flow

```
1. Upload
   POST /documents (multipart)
   → Validate file type + size
   → Generate storage key: users/{user_id}/{uuid}/{filename}
   → Upload raw file to MinIO
   → Insert document row (status=UPLOADED)
   → Publish DocumentUploaded event

2. Processing (triggered by event, runs in goroutine)
   → Download file from MinIO
   → Extract text (PDF: pdfcpu, DOCX: unioffice, TXT: direct)
   → Chunk: 512 tokens, 50-token overlap
   → For each chunk:
       - Call OpenAI embedding API
       - Upsert to Qdrant (point_id=chunk_id, payload={user_id, document_id, chunk_index})
       - Insert chunk row in Postgres
   → Update document status=INDEXED, chunk_count=N
   → Publish EmbeddingCreated event

3. Retrieval (called by chat service when rag_enabled=true)
   → Embed user query (same model)
   → Qdrant search: filter{user_id=X}, limit=20, score_threshold=0.5
   → Join with Postgres chunk rows (enrich with doc name)
   → Cross-encoder reranking (top-5)
   → Build context string with token budget (max 4000 tokens)
   → Inject as system context for the LLM
```

### Qdrant Collection Schema

```json
{
  "name": "documents",
  "vectors": { "size": 1536, "distance": "Cosine" },
  "payload_schema": {
    "user_id":     "keyword",
    "document_id": "keyword",
    "chunk_index": "integer"
  }
}
```

### Routes

```
POST   /api/v1/documents              Upload (multipart/form-data, field: file)
GET    /api/v1/documents              List user's documents
GET    /api/v1/documents/:id          Document metadata + status
DELETE /api/v1/documents/:id          Delete doc + chunks + Qdrant points
GET    /api/v1/documents/:id/chunks   List text chunks

POST   /api/v1/rag/search             Semantic search (body: {query, top_k, document_ids?})
```

---

## 13. Phase 4 — Memory

### What to Build

```
backend/internal/modules/memory/
├── application/port/
│   ├── memory_repository.go      # CRUD + FindByUserID with filters
│   └── memory_vector_port.go     # UpsertVector, Search interface
├── application/service/
│   ├── memory_service.go         # Create, List, Search, Delete, AutoExtract
│   └── extractor_service.go      # LLM-based memory extraction from chat
├── infrastructure/
│   ├── repository/memory_repo.go # Postgres
│   └── cache/memory_cache.go     # Redis short-term memory helpers
└── delivery/http/
    ├── handler.go
    └── routes.go
```

### Three-Layer Memory Architecture

| Layer | Storage | When Used | TTL |
|-------|---------|-----------|-----|
| **Short-term** | Redis | Last 20 messages per conversation | 2 hours sliding |
| **Long-term** | PostgreSQL | Explicit facts, preferences, events | Permanent (or user-set) |
| **Semantic** | Qdrant | Fuzzy "what do I know about X" lookups | Mirrors long-term |

### Auto-Extraction Flow

After each `ChatCompleted` event, the extractor service runs:

```go
// Use LLM to extract structured memories from the conversation turn:
systemPrompt := `Extract important personal information from this conversation.
Return JSON: [{"type": "FACT|PREFERENCE|EVENT|SKILL", "content": "...", "importance": 0.0-1.0}]
Only extract information worth remembering long-term.`

// Then upsert each extracted memory to Postgres + Qdrant
```

### Routes

```
GET    /api/v1/memories              List memories (filter: type, importance)
POST   /api/v1/memories              Create memory manually
PATCH  /api/v1/memories/:id          Update memory
DELETE /api/v1/memories/:id          Delete memory
POST   /api/v1/memories/search       Semantic search (body: {query, top_k})
```

---

## 14. Phase 5 — AI Agents (Eino)

### What to Build

```
backend/internal/modules/agent/
├── application/port/
│   ├── agent_repository.go
│   └── agent_orchestrator.go    # interface: Run(ctx, agent, messages) → stream
├── application/service/
│   ├── agent_service.go         # CRUD for agent definitions
│   └── agent_runner.go          # Builds Eino graph from Agent config, executes
├── infrastructure/
│   ├── repository/agent_repo.go
│   └── eino/
│       ├── supervisor_graph.go  # Eino: supervisor routes to sub-agents
│       ├── research_agent.go    # RAG + web search tools
│       ├── coding_agent.go      # GitHub tool + code explanation
│       ├── planning_agent.go    # Task breakdown + memory write
│       └── tool_node.go         # Wraps tool.Executor as Eino tool node
└── delivery/http/
    ├── handler.go
    └── routes.go
```

### Eino Graph Design

```go
// supervisor_graph.go
graph := compose.NewGraph[map[string]any, map[string]any]()

// Nodes
graph.AddLLMNode("supervisor", supervisorLLM)
graph.AddLambdaNode("router", routerFn)      // decides which sub-agent
graph.AddLambdaNode("research", researchAgent.Run)
graph.AddLambdaNode("coding", codingAgent.Run)
graph.AddLambdaNode("planning", planningAgent.Run)

// Edges
graph.AddEdge(compose.START, "supervisor")
graph.AddEdge("supervisor", "router")
graph.AddConditionalEdges("router", routeByType, map[string]string{
    "research": "research",
    "coding":   "coding",
    "planning": "planning",
    "done":     compose.END,
})

// Each sub-agent loops back to supervisor or exits
graph.AddEdge("research", "supervisor")
graph.AddEdge("coding",   "supervisor")
graph.AddEdge("planning", "supervisor")
```

### Before Each Agent Run

```go
// 1. Load short-term memory (Redis)
messages := loadShortTermMemory(conversationID)

// 2. Retrieve semantic memories (Qdrant)
memories := memoryService.Search(userID, userMessage, topK=5)

// 3. Retrieve RAG context (if rag_enabled)
ragContext := ragService.Search(userID, userMessage, topK=5)

// 4. Build system prompt with context injected
systemPrompt = buildSystemPrompt(agent, memories, ragContext)

// 5. Run Eino graph
result := einoGraph.Run(ctx, {messages, systemPrompt, tools})
```

### Routes

```
POST   /api/v1/agents               Create agent
GET    /api/v1/agents               List agents
GET    /api/v1/agents/:id           Get agent
PATCH  /api/v1/agents/:id           Update agent
DELETE /api/v1/agents/:id           Delete agent
POST   /api/v1/agents/:id/run       Run agent directly (test endpoint)
```

---

## 15. Phase 6 — Voice

### What to Build

```
backend/internal/modules/voice/
├── application/port/
│   └── transcription_port.go    # interface: Transcribe(audioKey) → text
├── application/service/
│   └── voice_service.go         # Upload audio → MinIO → Whisper → chat
├── infrastructure/
│   ├── storage/audio_storage.go # MinIO bucket: "audio"
│   └── transcription/
│       └── whisper.go           # OpenAI Whisper API call
└── delivery/http/
    ├── handler.go
    └── routes.go
```

### Voice Flow

```
POST /api/v1/voice/upload (multipart: audio file, conversation_id)
  → Validate: wav/mp3/m4a/webm, max 25MB
  → Upload to MinIO: audio/{user_id}/{uuid}.{ext}
  → Insert voice_session row (status=PENDING)
  → Enqueue transcription job (goroutine)
  → Return session ID

GET /api/v1/voice/sessions/:id   (poll for status + transcript)

// Background transcription:
  → Download audio from MinIO
  → POST to OpenAI Whisper API
  → Update voice_session (status=COMPLETED, transcript=...)
  → Inject transcript as user message into conversation
  → Run agent normally
  → Return AI response via WebSocket or polling
```

### Routes

```
POST   /api/v1/voice/upload          Upload audio file
GET    /api/v1/voice/sessions/:id    Get transcription status + result
GET    /api/v1/voice/sessions        List voice sessions
```

---

## 16. Phase 7 — Workflows

### What to Build

```
backend/internal/modules/workflow/
├── application/port/
│   ├── workflow_repository.go
│   └── workflow_runner.go        # interface: Run(workflow, payload)
├── application/service/
│   ├── workflow_service.go       # CRUD for workflow definitions
│   └── engine_service.go        # DAG executor: topological sort, parallel steps
├── infrastructure/
│   ├── repository/workflow_repo.go
│   └── engine/
│       ├── dag_executor.go       # Walks WorkflowDefinition graph
│       └── step_runner.go        # Routes node type → agent call / tool call / condition
└── delivery/http/
    ├── handler.go
    └── routes.go
```

### Workflow Definition (JSON)

```json
{
  "nodes": [
    { "id": "1", "type": "agent",  "config": { "agent_id": "uuid", "prompt": "Summarise {input}" } },
    { "id": "2", "type": "tool",   "config": { "tool": "email",   "to": "{email}", "body": "{output_1}" } },
    { "id": "3", "type": "condition", "config": { "if": "{output_1.length > 100}", "then": "2", "else": "END" } }
  ],
  "edges": [
    { "from": "START", "to": "1" },
    { "from": "1",     "to": "3" },
    { "from": "3",     "to": "2" }
  ],
  "trigger": { "type": "SCHEDULE", "cron_expr": "0 9 * * 1" }
}
```

### Routes

```
POST   /api/v1/workflows                  Create workflow
GET    /api/v1/workflows                  List workflows
GET    /api/v1/workflows/:id              Get workflow + definition
PATCH  /api/v1/workflows/:id             Update workflow
DELETE /api/v1/workflows/:id             Delete workflow
POST   /api/v1/workflows/:id/run         Trigger manual run
GET    /api/v1/workflows/:id/runs         List runs
GET    /api/v1/workflows/runs/:run_id     Get run status + result

POST   /api/v1/tools                     Register custom tool
GET    /api/v1/tools                     List available tools
PATCH  /api/v1/tools/:id/config          Configure user credentials for a tool
```

---

## 17. Phase 8 — Multi-Tenant

### What to Add

1. **Tenant table** — `tenants(id, name, plan, owner_id)`
2. **`tenant_id` enforcement** — every query adds `WHERE tenant_id = $1` via middleware
3. **Tenant isolation middleware**:

```go
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetHeader("X-Tenant-ID")
        // validate tenant membership
        c.Set("tenant_id", tenantID)
        c.Next()
    }
}
```

4. **Row-level security in Postgres** (optional, for strict isolation):

```sql
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON documents
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

5. **Tenant-scoped rate limits in Redis**: `rate:{tenant_id}:{user_id}`
6. **Per-tenant AI model config** — different models/budgets per plan

---

## 18. Coding Conventions

### Adding a New Module

Follow this exact sequence. The auth module is your reference implementation.

```
1. domain/entity/  — define the aggregate root, no external imports
2. domain/event/   — define domain events it emits
3. application/port/repository.go — define the interface
4. application/dto/ — request/response structs with validate tags
5. application/service/ — business logic, depends only on ports
6. infrastructure/repository/ — pgx implementation of the port
7. delivery/http/handler.go — bind request → service call → response
8. delivery/http/routes.go — mount routes with middleware
9. main.go — wire it all together (constructor injection)
```

### Error Handling

Always return `*apperrors.AppError`. Never return raw `error` from a service.

```go
// Good
if err != nil {
    return nil, apperrors.Internal(err)  // wraps cause, hides from client
}
if existing != nil {
    return nil, apperrors.Conflict("email already registered")
}

// Handler — just call response.Error(c, err)
res, err := s.authSvc.Register(ctx, req)
if err != nil {
    response.Error(c, err)  // unwraps AppError automatically
    return
}
```

### Response Format

```go
// Success
response.OK(c, data)               // 200
response.Created(c, data)          // 201
response.Paginated(c, data, meta)  // 200 with pagination meta
response.NoContent(c)              // 204

// Error — pass any error, AppError is unwrapped automatically
response.Error(c, err)
```

### Repository Pattern Rules

- Only the `infrastructure/repository` files import `pgxpool` or Redis
- Services depend on interfaces from `application/port`, never on concrete repos
- Never put business logic in a repository — it is pure CRUD

### Event Bus Rules

```go
// Publish asynchronously (default — non-blocking)
bus.Publish(ctx, event.UserRegistered{...})

// Publish synchronously (use in tests, or when side effects must complete)
bus.PublishSync(ctx, event.UserRegistered{...})

// Subscribe (wire in main.go or in a module's bootstrap func)
bus.Subscribe(event.EvtUserRegistered, auditModule.OnUserRegistered)
```

### Context Usage

Always pass `ctx context.Context` as the first argument to every service and repository method. The context carries the request deadline and cancellation signal.

---

## 19. Security Model

### JWT Strategy

```
Access Token:
  - Algorithm: HS256
  - Expiry: 15 minutes
  - Claims: sub (user ID), email, role
  - Sent: Authorization: Bearer header
  - Never stored anywhere

Refresh Token:
  - Format: two UUID v4s concatenated (72 chars of entropy)
  - Storage: SHA-256 hash only (never raw value in DB)
  - Transport: HttpOnly, Secure, SameSite=Strict cookie
  - Expiry: 7 days
  - Rotation: every refresh call revokes old + issues new
  - Revocation: instant (mark revoked=true in DB)
```

### OAuth CSRF Protection

```
1. Server generates UUID state
2. Stores state in Redis: key="oauth:state:{state}", TTL=10min
3. Returns state in AuthCodeURL
4. On callback: verify state exists in Redis, then DELETE it
5. If missing/expired → reject with 400
```

### Password Hashing

bcrypt with cost 10 (verified every login — ~100ms deliberate delay).

### Rate Limiting (to add in Phase 2)

```go
// Redis sliding window
// Key: rate:{ip}  or  rate:user:{user_id}
// Limit: 100 req/min per IP, 1000 req/min per user
```

### Tool Credentials Encryption (Phase 5)

```go
// Encrypt before storage:
encrypted := aes256GCM.Encrypt(key=masterKey, plaintext=json.Marshal(config))
// Store encrypted bytes in user_tool_configs.config

// Decrypt on use:
plain := aes256GCM.Decrypt(key=masterKey, ciphertext=config)
```

---

## 20. Microservice Extraction Plan

The module boundary is the extraction unit. Each module becomes one service.

### Extraction Order (by traffic pressure)

| Order | Module | Why This First | What Changes |
|-------|--------|---------------|-------------|
| 1 | `document` + `rag` | I/O heavy, CPU for chunking/embedding | Move folder, add gRPC server, swap event bus calls for NATS publish |
| 2 | `voice` | GPU for Whisper, latency sensitive | Same + GPU node pool in K8s |
| 3 | `workflow` | Stateful, long-running, needs its own queue | Add Temporal or internal job queue |
| 4 | `billing` | Compliance isolation, SOC2 requirement | Separate DB, separate deploy |
| 5 | `agent` | Compute heavy, needs independent scaling | Multiple replicas behind LB |

### What Each Extraction Requires

```
1. Move backend/internal/modules/{name}/ → new repo

2. Replace event bus Publish with NATS/Kafka:
   bus.Publish(ctx, event)
   → natsClient.Publish(event.EventName(), json.Marshal(event))

3. Replace direct service calls with gRPC or HTTP:
   ragService.Search(ctx, query)
   → ragClient.Search(ctx, &pb.SearchRequest{...})

4. Keep the same repository pattern — only the DB connection changes

5. Update docker-compose.yml / K8s manifests to deploy separately
```

### Database Split

When extracting, each service gets its own Postgres schema or cluster:

```sql
-- Before extraction: all tables in public schema
-- After: document service owns its schema
CREATE SCHEMA documents;
-- Move tables: documents.*, document_chunks.*

-- Cross-service queries become API calls, never joins
```

---

## Quick Reference

```bash
# Start dev
make setup && make dev && make dev-front

# Test auth flow
curl -X POST :8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@test.com","password":"password123","full_name":"Dev User"}'

# Verify token
curl :8080/api/v1/auth/me \
  -H 'Authorization: Bearer <access_token>'

# Build + run tests
make build && make test

# Apply new migrations
make migrate-up

# Rebuild Docker images
make docker-build && make docker-up
```

---

*Last updated: All 8 phases complete. Platform is production-ready.*
