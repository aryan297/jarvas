package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarvas/backend/internal/modules/tenant/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// ── Tenant repo ───────────────────────────────────────────────────────────────

type pgTenantRepository struct{ db *pgxpool.Pool }

func NewTenantRepository(db *pgxpool.Pool) *pgTenantRepository {
	return &pgTenantRepository{db: db}
}

func (r *pgTenantRepository) Save(ctx context.Context, t *entity.Tenant) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, plan, owner_id, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		t.ID, t.Name, t.Slug, string(t.Plan), t.OwnerID, t.IsActive, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *pgTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, plan, owner_id, is_active, created_at, updated_at
		FROM tenants WHERE id=$1 AND is_active=TRUE`, id)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("tenant")
		}
		return nil, err
	}
	return t, nil
}

func (r *pgTenantRepository) FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Tenant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, plan, owner_id, is_active, created_at, updated_at
		FROM tenants WHERE owner_id=$1 AND is_active=TRUE ORDER BY created_at ASC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*entity.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err == nil {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *pgTenantRepository) Update(ctx context.Context, t *entity.Tenant) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tenants SET name=$2, slug=$3, plan=$4, updated_at=$5 WHERE id=$1`,
		t.ID, t.Name, t.Slug, string(t.Plan), t.UpdatedAt,
	)
	return err
}

// ── Member repo ───────────────────────────────────────────────────────────────

type pgMemberRepository struct{ db *pgxpool.Pool }

func NewMemberRepository(db *pgxpool.Pool) *pgMemberRepository {
	return &pgMemberRepository{db: db}
}

func (r *pgMemberRepository) AddMember(ctx context.Context, m *entity.TenantMember) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenant_members (id, tenant_id, user_id, role, invited_by, joined_at)
		VALUES ($1,$2,$3,$4::tenant_role,$5,$6)
		ON CONFLICT (tenant_id, user_id) DO NOTHING`,
		m.ID, m.TenantID, m.UserID, string(m.Role), m.InvitedBy, m.JoinedAt,
	)
	return err
}

func (r *pgMemberRepository) FindMember(ctx context.Context, tenantID, userID uuid.UUID) (*entity.TenantMember, error) {
	var m entity.TenantMember
	var role string
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, role::text, invited_by, joined_at
		FROM tenant_members WHERE tenant_id=$1 AND user_id=$2`,
		tenantID, userID,
	).Scan(&m.ID, &m.TenantID, &m.UserID, &role, &m.InvitedBy, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("tenant member")
		}
		return nil, err
	}
	m.Role = entity.TenantRole(role)
	return &m, nil
}

func (r *pgMemberRepository) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]*entity.TenantMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tm.id, tm.tenant_id, tm.user_id, tm.role::text, tm.invited_by, tm.joined_at,
		       u.email, u.full_name
		FROM tenant_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id=$1
		ORDER BY tm.joined_at ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*entity.TenantMember
	for rows.Next() {
		var m entity.TenantMember
		var role string
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.UserID, &role, &m.InvitedBy, &m.JoinedAt,
			&m.UserEmail, &m.UserFullName,
		); err == nil {
			m.Role = entity.TenantRole(role)
			result = append(result, &m)
		}
	}
	return result, nil
}

func (r *pgMemberRepository) ListTenantsByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Tenant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.plan, t.owner_id, t.is_active, t.created_at, t.updated_at
		FROM tenants t
		JOIN tenant_members tm ON tm.tenant_id = t.id
		WHERE tm.user_id=$1 AND t.is_active=TRUE
		ORDER BY t.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*entity.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err == nil {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *pgMemberRepository) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM tenant_members WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID)
	return err
}

func (r *pgMemberRepository) FindUserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, email).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, apperrors.NotFound("user with that email")
		}
		return uuid.Nil, err
	}
	return id, nil
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

type scanner interface{ Scan(dest ...any) error }

func scanTenant(row scanner) (*entity.Tenant, error) {
	var t entity.Tenant
	var plan string
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &plan, &t.OwnerID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Plan = entity.TenantPlan(plan)
	return &t, nil
}

// slugify converts a name to a lowercase, hyphenated slug with a short UUID suffix.
func Slugify(name string, id uuid.UUID) string {
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32)
		} else if r == ' ' || r == '-' || r == '_' {
			slug += "-"
		}
	}
	// Append first 8 chars of UUID to guarantee uniqueness.
	shortID := id.String()[:8]
	return slug + "-" + shortID
}

// unused import guard
var _ = time.Now
