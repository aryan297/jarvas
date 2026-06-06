package dto

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role"  binding:"omitempty,oneof=ADMIN MEMBER"`
}

type TenantResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Plan      string `json:"plan"`
	OwnerID   string `json:"owner_id"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type MemberResponse struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	UserEmail    string `json:"user_email"`
	UserFullName string `json:"user_full_name"`
	Role         string `json:"role"`
	JoinedAt     string `json:"joined_at"`
}
