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
	"github.com/jarvas/backend/internal/modules/auth/application/service"
	authinfrarepo "github.com/jarvas/backend/internal/modules/auth/infrastructure/repository"
	"github.com/jarvas/backend/internal/modules/auth/infrastructure/oauth"
	authhttp "github.com/jarvas/backend/internal/modules/auth/delivery/http"
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

	// ── Infrastructure ────────────────────────────────────────────────────────

	ctx := context.Background()

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

	// ── Module: Auth ─────────────────────────────────────────────────────────

	userRepo  := authinfrarepo.NewUserRepository(db.Pool)
	tokenRepo := authinfrarepo.NewTokenRepository(db.Pool)
	tokenSvc  := service.NewTokenService(cfg.JWT)
	authSvc   := service.NewAuthService(userRepo, tokenRepo, tokenSvc, bus)
	googleProvider := oauth.NewGoogleProvider(cfg.Google)
	authHandler    := authhttp.NewAuthHandler(authSvc, googleProvider, redisClient)

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

	// ── Health check ──────────────────────────────────────────────────────────

	router.GET("/health", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": cfg.App.Name})
	})

	// ── API v1 ────────────────────────────────────────────────────────────────

	v1 := router.Group("/api/v1")
	authhttp.RegisterRoutes(v1, authHandler, authMW)

	// Additional module routes are registered here as they are implemented:
	// userhttp.RegisterRoutes(v1, userHandler, authMW)
	// chathttp.RegisterRoutes(v1, chatHandler, authMW)
	// documenthttp.RegisterRoutes(v1, documentHandler, authMW)
	// raghttp.RegisterRoutes(v1, ragHandler, authMW)
	// memoryhttp.RegisterRoutes(v1, memoryHandler, authMW)
	// agenthttp.RegisterRoutes(v1, agentHandler, authMW)
	// workflowhttp.RegisterRoutes(v1, workflowHandler, authMW)
	// toolhttp.RegisterRoutes(v1, toolHandler, authMW)
	// voicehttp.RegisterRoutes(v1, voiceHandler, authMW)

	// ── HTTP Server with graceful shutdown ────────────────────────────────────

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
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

	// Suppress unused variable warning for bus — it will be used by event subscribers.
	_ = bus
}
