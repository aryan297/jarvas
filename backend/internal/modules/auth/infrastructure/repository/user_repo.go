package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type pgUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *pgUserRepository {
	return &pgUserRepository{db: db}
}

func (r *pgUserRepository) Create(ctx context.Context, u *entity.User) error {
	// Explicit ::enum casts required — pgx v5 sends Go strings as text and
	// Postgres does not implicitly coerce text → custom enum in parameterised queries.
	// nullStr: empty string → NULL so the partial unique index on provider_id works correctly.
	q := `
		INSERT INTO users (id, email, password_hash, full_name, avatar_url,
		                   role, status, provider, provider_id, email_verified,
		                   created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,
		        $6::user_role, $7::user_status, $8::auth_provider,
		        $9,$10,$11,$12)`

	_, err := r.db.Exec(ctx, q,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.AvatarURL,
		string(u.Role), string(u.Status), string(u.Provider),
		nullStr(u.ProviderID), u.EmailVerified,
		u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (r *pgUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	q := `SELECT id, email, password_hash, full_name, avatar_url,
	             role, status, provider, provider_id, email_verified,
	             last_login_at, created_at, updated_at
	      FROM users WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *pgUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	q := `SELECT id, email, password_hash, full_name, avatar_url,
	             role, status, provider, provider_id, email_verified,
	             last_login_at, created_at, updated_at
	      FROM users WHERE email = $1`
	return r.scanOne(ctx, q, email)
}

func (r *pgUserRepository) FindByProviderID(ctx context.Context, provider, providerID string) (*entity.User, error) {
	q := `SELECT id, email, password_hash, full_name, avatar_url,
	             role, status, provider, provider_id, email_verified,
	             last_login_at, created_at, updated_at
	      FROM users WHERE provider = $1::auth_provider AND provider_id = $2`
	return r.scanOne(ctx, q, provider, providerID)
}

func (r *pgUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *pgUserRepository) Update(ctx context.Context, u *entity.User) error {
	q := `UPDATE users
	      SET email=$2, full_name=$3, avatar_url=$4,
	          role=$5::user_role, status=$6::user_status,
	          provider=$7::auth_provider, provider_id=$8,
	          email_verified=$9, updated_at=NOW()
	      WHERE id=$1`
	_, err := r.db.Exec(ctx, q,
		u.ID, u.Email, u.FullName, u.AvatarURL,
		string(u.Role), string(u.Status), string(u.Provider),
		u.ProviderID, u.EmailVerified,
	)
	return err
}

// nullStr converts an empty string to nil so Postgres stores NULL.
// Required for nullable VARCHAR columns with partial unique indexes.
func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *pgUserRepository) scanOne(ctx context.Context, query string, args ...interface{}) (*entity.User, error) {
	row := r.db.QueryRow(ctx, query, args...)
	var u entity.User
	// Nullable columns use pointer scan vars; enum columns scan as plain strings.
	var passwordHash, avatarURL, providerID *string
	var role, status, provider string

	err := row.Scan(
		&u.ID, &u.Email, &passwordHash, &u.FullName, &avatarURL,
		&role, &status, &provider,
		&providerID, &u.EmailVerified,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("user")
		}
		return nil, err
	}

	u.Role = entity.UserRole(role)
	u.Status = entity.UserStatus(status)
	u.Provider = entity.AuthProvider(provider)
	if passwordHash != nil {
		u.PasswordHash = *passwordHash
	}
	if avatarURL != nil {
		u.AvatarURL = *avatarURL
	}
	if providerID != nil {
		u.ProviderID = *providerID
	}
	return &u, nil
}
