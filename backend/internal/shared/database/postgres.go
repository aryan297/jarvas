package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/shared/config"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

// Pool wraps pgxpool.Pool for dependency injection and lifecycle management.
type Pool struct {
	*pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, cfg config.DBConfig) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("postgres connected", zap.String("host", cfg.Host), zap.String("db", cfg.Name))
	return &Pool{pool}, nil
}
