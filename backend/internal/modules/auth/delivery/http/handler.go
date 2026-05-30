package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/auth/application/dto"
	"github.com/jarvas/backend/internal/modules/auth/application/service"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	"github.com/jarvas/backend/internal/modules/auth/infrastructure/oauth"
	"github.com/jarvas/backend/internal/shared/cache"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
	"github.com/jarvas/backend/internal/shared/validator"
)

const (
	refreshTokenCookie = "refresh_token"
	oauthStateTTL      = 10 * time.Minute
)

type AuthHandler struct {
	authSvc *service.AuthService
	google  *oauth.GoogleProvider
	cache   *cache.Client
}

func NewAuthHandler(authSvc *service.AuthService, google *oauth.GoogleProvider, cache *cache.Client) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, google: google, cache: cache}
}

// Register godoc
// @Summary     Register a new user
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.RegisterRequest true "Registration payload"
// @Success     201  {object} dto.AuthResponse
// @Failure     400  {object} response.errorBody
// @Failure     409  {object} response.errorBody
// @Router      /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	if err := validator.Validate(&req); err != nil {
		response.Error(c, err)
		return
	}

	res, err := h.authSvc.Register(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	response.Created(c, res)
}

// Login godoc
// @Summary     Login with email and password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.LoginRequest true "Login payload"
// @Success     200  {object} dto.AuthResponse
// @Failure     401  {object} response.errorBody
// @Router      /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	if err := validator.Validate(&req); err != nil {
		response.Error(c, err)
		return
	}

	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	res, err := h.authSvc.Login(c.Request.Context(), req, ip, ua)
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	response.OK(c, res)
}

// Refresh godoc
// @Summary     Refresh access token
// @Tags        auth
// @Produce     json
// @Success     200 {object} dto.TokenPair
// @Failure     401 {object} response.errorBody
// @Router      /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	rawToken, err := c.Cookie(refreshTokenCookie)
	if err != nil || rawToken == "" {
		// Fall back to body for non-browser clients.
		var req dto.RefreshRequest
		if err2 := c.ShouldBindJSON(&req); err2 != nil || req.RefreshToken == "" {
			response.Error(c, apperrors.Unauthorized("refresh token missing"))
			return
		}
		rawToken = req.RefreshToken
	}

	pair, err := h.authSvc.RefreshTokens(c.Request.Context(), rawToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, pair.RefreshToken)
	response.OK(c, pair)
}

// Logout godoc
// @Summary     Logout and revoke refresh token
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     204
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	rawToken, _ := c.Cookie(refreshTokenCookie)
	userID, _ := c.Get("user_id")

	if rawToken != "" {
		_ = h.authSvc.Logout(c.Request.Context(), rawToken, userID.(string))
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	response.NoContent(c)
}

// GoogleLogin godoc
// @Summary     Start Google OAuth flow
// @Tags        auth
// @Produce     json
// @Success     200 {object} dto.GoogleOAuthURLResponse
// @Router      /auth/google/login [get]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state := uuid.New().String()
	// Store state in Redis to prevent CSRF.
	_ = h.cache.SetString(c.Request.Context(), "oauth:state:"+state, "valid", oauthStateTTL)
	url := h.google.AuthCodeURL(state)
	response.OK(c, dto.GoogleOAuthURLResponse{URL: url})
}

// GoogleCallback godoc
// @Summary     Google OAuth callback
// @Tags        auth
// @Produce     json
// @Param       code  query string true "Authorization code"
// @Param       state query string true "CSRF state"
// @Success     200 {object} dto.AuthResponse
// @Failure     400 {object} response.errorBody
// @Router      /auth/google/callback [get]
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		response.Error(c, apperrors.BadRequest("missing state or code"))
		return
	}

	// Validate CSRF state.
	stateKey := "oauth:state:" + state
	exists, err := h.cache.Exists(c.Request.Context(), stateKey)
	if err != nil || !exists {
		response.Error(c, apperrors.BadRequest("invalid or expired oauth state"))
		return
	}
	_ = h.cache.Del(c.Request.Context(), stateKey)

	userInfo, err := h.google.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		response.Error(c, apperrors.Internal(err))
		return
	}

	res, err := h.authSvc.HandleOAuthUser(
		c.Request.Context(),
		userInfo.Email,
		userInfo.Name,
		userInfo.Sub,
		entity.ProviderGoogle,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	response.OK(c, res)
}

// Me godoc
// @Summary     Get current authenticated user
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.UserResponse
// @Failure     401 {object} response.errorBody
// @Router      /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	// Middleware sets user_id, user_email, user_role — not a "user" object.
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized("not authenticated"))
		return
	}
	response.OK(c, dto.UserResponse{
		ID:    userID.(string),
		Email: c.GetString("user_email"),
		Role:  string(c.MustGet("user_role").(entity.UserRole)),
	})
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    token,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}
