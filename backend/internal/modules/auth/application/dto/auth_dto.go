package dto

// RegisterRequest is the payload for email/password registration.
type RegisterRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8,max=72"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
}

// LoginRequest is the payload for email/password login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// TokenPair is returned on successful auth. Access token is short-lived;
// refresh token is long-lived and stored as an HTTP-only cookie.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	TokenType    string `json:"token_type"`
}

// RefreshRequest carries the refresh token for rotation.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UserResponse is the public-facing user representation.
type UserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	FullName      string  `json:"full_name"`
	AvatarURL     string  `json:"avatar_url,omitempty"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
}

// GoogleOAuthURLResponse carries the authorization URL to redirect the user.
type GoogleOAuthURLResponse struct {
	URL string `json:"url"`
}

// AuthResponse bundles tokens + user profile for the login/register response.
type AuthResponse struct {
	TokenPair
	User UserResponse `json:"user"`
}
