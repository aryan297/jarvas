package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// Auth
	authhttp      "github.com/jarvas/backend/internal/modules/auth/delivery/http"
	"github.com/jarvas/backend/internal/modules/auth/infrastructure/oauth"
	authinfrarepo "github.com/jarvas/backend/internal/modules/auth/infrastructure/repository"
	authsvc       "github.com/jarvas/backend/internal/modules/auth/application/service"

	// Chat
	chathttp  "github.com/jarvas/backend/internal/modules/chat/delivery/http"
	chatevent "github.com/jarvas/backend/internal/modules/chat/domain/event"
	chatrepo  "github.com/jarvas/backend/internal/modules/chat/infrastructure/repository"
	chatsvc   "github.com/jarvas/backend/internal/modules/chat/application/service"

	// Document
	dochttp    "github.com/jarvas/backend/internal/modules/document/delivery/http"
	docrepo    "github.com/jarvas/backend/internal/modules/document/infrastructure/repository"
	docstorage "github.com/jarvas/backend/internal/modules/document/infrastructure/storage"
	docsvc     "github.com/jarvas/backend/internal/modules/document/application/service"
	docevent   "github.com/jarvas/backend/internal/modules/document/domain/event"

	// RAG
	raghttp      "github.com/jarvas/backend/internal/modules/rag/delivery/http"
	ragembedding "github.com/jarvas/backend/internal/modules/rag/infrastructure/embedding"
	ragvector    "github.com/jarvas/backend/internal/modules/rag/infrastructure/vectorstore"
	ragsvc       "github.com/jarvas/backend/internal/modules/rag/application/service"

	// Memory
	memhttp "github.com/jarvas/backend/internal/modules/memory/delivery/http"
	memrepo "github.com/jarvas/backend/internal/modules/memory/infrastructure/repository"
	memsvc  "github.com/jarvas/backend/internal/modules/memory/application/service"

	// Agent
	agenthttp "github.com/jarvas/backend/internal/modules/agent/delivery/http"
	agentrepo "github.com/jarvas/backend/internal/modules/agent/infrastructure/repository"
	agentsvc  "github.com/jarvas/backend/internal/modules/agent/application/service"

	// Tools
	toolsvc       "github.com/jarvas/backend/internal/modules/tool/application/service"
	toolexecutors "github.com/jarvas/backend/internal/modules/tool/infrastructure/executors"

	// Voice
	voicehttp        "github.com/jarvas/backend/internal/modules/voice/delivery/http"
	voicecache       "github.com/jarvas/backend/internal/modules/voice/infrastructure/cache"
	voicestorage     "github.com/jarvas/backend/internal/modules/voice/infrastructure/storage"
	voicetranscript  "github.com/jarvas/backend/internal/modules/voice/infrastructure/transcription"
	voicesvc         "github.com/jarvas/backend/internal/modules/voice/application/service"

	// Shared
	"github.com/jarvas/backend/internal/shared/cache"
	"github.com/jarvas/backend/internal/shared/config"
	"github.com/jarvas/backend/internal/shared/database"
	"github.com/jarvas/backend/internal/shared/eventbus"
	"github.com/jarvas/backend/internal/shared/logger"
	"github.com/jarvas/backend/internal/shared/middleware"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	logger.Init(cfg.App.Env)
	defer logger.Sync()

	ctx := context.Background()

	// ── Infrastructure ────────────────────────────────────────────────────────

	db, err := database.NewPostgresPool(ctx, cfg.DB)
	if err != nil {
		logger.Fatal("postgres connection failed", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("redis connection failed", zap.Error(err))
	}
	defer redisClient.Close()

	bus := eventbus.New()

	// ── Module: Auth ──────────────────────────────────────────────────────────

	userRepo       := authinfrarepo.NewUserRepository(db.Pool)
	tokenRepo      := authinfrarepo.NewTokenRepository(db.Pool)
	tokenSvc       := authsvc.NewTokenService(cfg.JWT)
	authService    := authsvc.NewAuthService(userRepo, tokenRepo, tokenSvc, bus)
	googleProvider := oauth.NewGoogleProvider(cfg.Google)
	authHandler    := authhttp.NewAuthHandler(authService, googleProvider, redisClient)

	// ── Module: Chat ──────────────────────────────────────────────────────────

	convRepo    := chatrepo.NewConversationRepository(db.Pool)
	msgRepo     := chatrepo.NewMessageRepository(db.Pool)
	chatService := chatsvc.NewChatService(convRepo, msgRepo, redisClient, bus, chatsvc.ChatConfig{
		OpenAIKey: cfg.AI.OpenAIKey,
		Model:     cfg.AI.Model,
		MaxTokens: 4096,
	})
	chatHandler := chathttp.NewChatHandler(chatService)

	// ── Module: Document + RAG ────────────────────────────────────────────────

	minioStorage, err := docstorage.NewMinIOStorage(cfg.MinIO)
	if err != nil {
		logger.Fatal("minio connection failed", zap.Error(err))
	}

	qdrantStore, err := ragvector.NewQdrantStore(cfg.Qdrant.Host, cfg.Qdrant.Port, cfg.Qdrant.APIKey)
	if err != nil {
		logger.Fatal("qdrant connection failed", zap.Error(err))
	}

	embedder := ragembedding.NewOpenAIEmbedder(cfg.AI.OpenAIKey, cfg.AI.EmbeddingModel)

	docRepository   := docrepo.NewDocumentRepository(db.Pool)
	chunkRepository := docrepo.NewChunkRepository(db.Pool)

	processor := ragsvc.NewProcessor(docRepository, chunkRepository, minioStorage, embedder, qdrantStore, bus)
	if err := processor.EnsureCollection(ctx); err != nil {
		logger.Fatal("qdrant ensure collection (documents) failed", zap.Error(err))
	}

	documentService := docsvc.NewDocumentService(docRepository, chunkRepository, minioStorage, bus)
	documentHandler := dochttp.NewDocumentHandler(documentService)

	ragService := ragsvc.NewRAGService(embedder, qdrantStore)
	ragHandler  := raghttp.NewRAGHandler(ragService)

	// ── Module: Memory ────────────────────────────────────────────────────────

	memRepository := memrepo.NewMemoryRepository(db.Pool)
	memoryService := memsvc.NewMemoryService(
		memRepository, embedder, qdrantStore,
		cfg.AI.OpenAIKey, cfg.AI.Model,
	)
	if err := memoryService.EnsureCollection(ctx); err != nil {
		logger.Fatal("qdrant ensure collection (memory) failed", zap.Error(err))
	}
	chatService.SetMemoryRetriever(memoryService)
	memoryHandler := memhttp.NewMemoryHandler(memoryService)

	// ── Module: Tools ─────────────────────────────────────────────────────────

	toolRegistry := toolsvc.NewRegistry()
	toolRegistry.Register(toolexecutors.WebSearchDef())
	toolRegistry.Register(toolexecutors.CalculatorDef())

	// ── Module: Agent ─────────────────────────────────────────────────────────

	agentRepository := agentrepo.NewAgentRepository(db.Pool)
	agentService    := agentsvc.NewAgentService(agentRepository)
	runnerService   := agentsvc.NewRunnerService(agentRepository, toolRegistry, cfg.AI.OpenAIKey)
	runnerService.SetMemoryRetriever(memoryService)

	// Wire agent runner into chat so agent conversations use Eino.
	chatService.SetAgentRunner(runnerService)

	agentHandler := agenthttp.NewAgentHandler(agentService)

	// ── Module: Voice ─────────────────────────────────────────────────────────

	audioStorage, err := voicestorage.NewMinIOAudioStorage(cfg.MinIO)
	if err != nil {
		logger.Fatal("minio audio storage failed", zap.Error(err))
	}

	whisper       := voicetranscript.NewWhisperTranscriber(cfg.AI.OpenAIKey)
	sessionStore  := voicecache.NewRedisSessionStore(redisClient)
	voiceService  := voicesvc.NewVoiceService(sessionStore, audioStorage, whisper)
	voiceHandler  := voicehttp.NewVoiceHandler(voiceService)

	// ── Event subscriptions ───────────────────────────────────────────────────

	bus.Subscribe(docevent.EvtDocumentUploaded, func(ctx context.Context, e eventbus.Event) {
		evt := e.(docevent.DocumentUploaded)
		go processor.ProcessDocument(ctx, evt.DocumentID, evt.UserID)
	})

	bus.Subscribe(docevent.EvtDocumentDeleted, func(ctx context.Context, e eventbus.Event) {
		evt := e.(docevent.DocumentDeleted)
		go func() {
			_ = qdrantStore.DeleteByFilter(ctx, ragsvc.CollectionDocuments, map[string]string{
				"document_id": evt.DocumentID.String(),
			})
		}()
	})

	bus.Subscribe(chatevent.EvtChatCompleted, func(ctx context.Context, e eventbus.Event) {
		evt := e.(chatevent.ChatCompleted)
		go func() {
			_ = memoryService.Extract(ctx, evt.UserID, evt.UserMsg, evt.AssistMsg)
		}()
	})

	// ── Middleware ────────────────────────────────────────────────────────────

	authMW := middleware.NewAuthMiddleware(tokenSvc)

	// ── Gin Router ────────────────────────────────────────────────────────────

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.Origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Routes ────────────────────────────────────────────────────────────────

	router.GET("/health", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": cfg.App.Name})
	})

	v1 := router.Group("/api/v1")
	authhttp.RegisterRoutes(v1, authHandler, authMW)
	chathttp.RegisterRoutes(v1, chatHandler, authMW)
	dochttp.RegisterRoutes(v1, documentHandler, authMW)
	raghttp.RegisterRoutes(v1, ragHandler, authMW)
	memhttp.RegisterRoutes(v1, memoryHandler, authMW)
	agenthttp.RegisterRoutes(v1, agentHandler, authMW)
	voicehttp.RegisterRoutes(v1, voiceHandler, authMW)

	// ── HTTP Server ───────────────────────────────────────────────────────────

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("port", cfg.App.Port), zap.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("server stopped cleanly")
	}
}
