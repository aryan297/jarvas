package dto

type UpdateProfileRequest struct {
	FullName    string `json:"full_name" validate:"omitempty,min=2,max=100"`
	Bio         string `json:"bio"        validate:"omitempty,max=500"`
	AvatarURL   string `json:"avatar_url" validate:"omitempty,url"`
}

type UpdatePreferencesRequest struct {
	Theme         string `json:"theme"          validate:"omitempty,oneof=light dark system"`
	Language      string `json:"language"       validate:"omitempty,min=2,max=10"`
	Timezone      string `json:"timezone"       validate:"omitempty,max=50"`
	Notifications bool   `json:"notifications"`
}

type UserDetailResponse struct {
	ID            string            `json:"id"`
	Email         string            `json:"email"`
	FullName      string            `json:"full_name"`
	AvatarURL     string            `json:"avatar_url,omitempty"`
	Bio           string            `json:"bio,omitempty"`
	Role          string            `json:"role"`
	Plan          string            `json:"plan"`
	Preferences   map[string]interface{} `json:"preferences"`
	EmailVerified bool              `json:"email_verified"`
	CreatedAt     string            `json:"created_at"`
}
