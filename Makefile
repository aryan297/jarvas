.PHONY: help dev dev-noair install-air build test lint migrate-up migrate-down \
        docker-up docker-down docker-build seed clean fmt vet tidy

# ── Variables ─────────────────────────────────────────────────────────────────
BINARY    := server
BUILD_DIR := backend/bin
GO        := go
GOFLAGS   := -ldflags="-w -s"
DOCKER    := docker compose
AIR       := $(shell which air 2>/dev/null)

# Auto-create .env from example if it doesn't exist.
ENV_FILE := .env
$(ENV_FILE):
	@echo "⚠️  .env not found — copying from .env.example"
	@cp .env.example .env
	@echo "✅ .env created. Edit it to set JWT_SECRET, GOOGLE_CLIENT_ID, OPENAI_API_KEY, etc."

# ── Help ──────────────────────────────────────────────────────────────────────
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Development ───────────────────────────────────────────────────────────────
dev: $(ENV_FILE) infra ## Start backend (air if installed, else go run). Starts infra automatically.
	@if [ -n "$(AIR)" ]; then \
	  echo "🔄 Starting with air hot-reload..."; \
	  cd backend && air -c .air.toml; \
	else \
	  echo "ℹ️  air not found — running with 'go run' (no hot-reload)"; \
	  echo "   To get hot-reload: make install-air"; \
	  cd backend && $(GO) run ./cmd/server; \
	fi

dev-front: ## Start frontend dev server
	cd frontend && npm run dev

infra: $(ENV_FILE) ## Start only infrastructure (Postgres, Redis, Qdrant, MinIO)
	@docker info > /dev/null 2>&1 || { \
	  echo "🚀 Docker is not running. Starting Docker Desktop..."; \
	  open -a Docker; \
	  echo "⏳ Waiting for Docker to start (up to 30s)..."; \
	  for i in 1 2 3 4 5 6 7 8 9 10; do \
	    sleep 3; \
	    docker info > /dev/null 2>&1 && echo "✅ Docker is ready" && break; \
	    echo "   still waiting... ($$i/10)"; \
	  done; \
	  docker info > /dev/null 2>&1 || { echo "❌ Docker did not start. Open Docker Desktop manually and retry."; exit 1; }; \
	}
	@echo "🐳 Starting infrastructure services..."
	$(DOCKER) up -d postgres redis qdrant minio
	@echo "⏳ Waiting for Postgres to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
	  $(DOCKER) exec jarvas_postgres pg_isready -U jarvas -d jarvas_db -q 2>/dev/null && echo "✅ Postgres ready" && break; \
	  echo "   waiting... ($$i/10)"; sleep 2; \
	done

install-air: ## Install air hot-reload tool
	@echo "Installing air..."
	go install github.com/air-verse/air@latest
	@echo "✅ air installed. Run 'make dev' again."

# ── Build ─────────────────────────────────────────────────────────────────────
build: ## Build the backend binary
	mkdir -p $(BUILD_DIR)
	cd backend && $(GO) build $(GOFLAGS) -o ../$(BUILD_DIR)/$(BINARY) ./cmd/server

# ── Test ──────────────────────────────────────────────────────────────────────
test: ## Run all backend unit tests
	cd backend && $(GO) test ./... -v -race -cover -coverprofile=coverage.out

test-ci: ## Run tests with XML output (for CI)
	cd backend && $(GO) test ./... -race -coverprofile=coverage.out

# ── Code Quality ──────────────────────────────────────────────────────────────
lint: ## Lint with golangci-lint
	cd backend && golangci-lint run ./...

fmt: ## Format Go code
	cd backend && $(GO) fmt ./...

vet: ## Run go vet
	cd backend && $(GO) vet ./...

tidy: ## Tidy go modules
	cd backend && $(GO) mod tidy

# ── Database ──────────────────────────────────────────────────────────────────
migrate-up: ## Run all pending migrations
	@echo "Running migrations..."
	$(DOCKER) exec -T jarvas_postgres psql -U jarvas -d jarvas_db \
	  -f /docker-entrypoint-initdb.d/000_init.sql 2>/dev/null || true
	@for f in backend/migrations/*.sql; do \
	  echo "Applying $$f..."; \
	  $(DOCKER) exec -T jarvas_postgres psql -U jarvas -d jarvas_db < $$f || true; \
	done
	@echo "Done."

migrate-create: ## Create a new migration file: make migrate-create NAME=add_indexes
	@touch backend/migrations/$(shell date +%03d)_$(NAME).sql
	@echo "Created: backend/migrations/$(shell date +%03d)_$(NAME).sql"

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up: ## Start all Docker services
	cp -n .env.example .env 2>/dev/null || true
	$(DOCKER) up -d

docker-down: ## Stop all Docker services
	$(DOCKER) down

docker-build: ## Rebuild Docker images
	$(DOCKER) build

docker-logs: ## Tail logs from all services
	$(DOCKER) logs -f

docker-ps: ## Show running containers
	$(DOCKER) ps

# ── Frontend ──────────────────────────────────────────────────────────────────
front-install: ## Install frontend dependencies
	cd frontend && npm install

front-build: ## Build frontend for production
	cd frontend && npm run build

front-lint: ## Lint frontend code
	cd frontend && npm run lint

# ── Utilities ─────────────────────────────────────────────────────────────────
seed: ## Seed development data
	cd backend && $(GO) run ./scripts/seed/main.go

clean: ## Remove build artefacts
	rm -rf $(BUILD_DIR) backend/coverage.out frontend/dist

# ── Quick Setup ───────────────────────────────────────────────────────────────
setup: ## Full dev environment setup (first time)
	@echo "1. Copying .env.example → .env"
	cp -n .env.example .env || true
	@echo "2. Starting infrastructure..."
	$(DOCKER) up -d postgres redis qdrant minio
	@echo "3. Waiting for postgres..."
	@sleep 5
	@echo "4. Running migrations..."
	$(MAKE) migrate-up
	@echo "5. Installing frontend deps..."
	$(MAKE) front-install
	@echo ""
	@echo "Setup complete!"
	@echo "  Backend:  make dev"
	@echo "  Frontend: make dev-front"
	@echo "  All:      make docker-up"
