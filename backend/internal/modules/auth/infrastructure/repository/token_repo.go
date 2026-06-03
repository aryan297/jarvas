package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgTokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) *pgTokenRepository {
	return &pgTokenRepository{db: db}
}

func (r *pgTokenRepository) Save(ctx context.Context, t *entity.RefreshToken) error {
	q := `INSERT INTO refresh_tokens
	      (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.db.Exec(ctx, q,
		t.ID, t.UserID, t.TokenHash, nullStr(t.IPAddress), t.UserAgent, t.ExpiresAt, t.CreatedAt,
	)
	return err
}


func (r *pgTokenRepository) FindByHash(ctx context.Context, hash string) (*entity.RefreshToken, error) {
	// Cast ip_address::text — pgx v5 cannot scan INET directly into *string.
	q := `SELECT id, user_id, token_hash, ip_address::text, user_agent,
	             revoked, expires_at, created_at
	      FROM refresh_tokens WHERE token_hash = $1`
	row := r.db.QueryRow(ctx, q, hash)
	var t entity.RefreshToken
	// ip_address is INET (nullable); user_agent is TEXT (nullable) — use pointer vars.
	var ipAddress, userAgent *string
	err := row.Scan(
		&t.ID, &t.UserID, &t.TokenHash, &ipAddress, &userAgent,
		&t.Revoked, &t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("refresh token")
		}
		return nil, err
	}
	if ipAddress != nil {
		t.IPAddress = *ipAddress
	}
	if userAgent != nil {
		t.UserAgent = *userAgent
	}
	return &t, nil
}

func (r *pgTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=TRUE WHERE token_hash=$1`, hash)
	return err
}

func (r *pgTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=TRUE WHERE user_id=$1`, userID)
	return err
}

func (r *pgTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1`, time.Now().UTC())
	return err
}
