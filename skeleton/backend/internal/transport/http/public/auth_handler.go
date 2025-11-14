package public

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

// authProxy captures the delegated client surface the handler needs.
type authProxy interface {
	Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*iamservice.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
}

// AuthHandler exposes /api/v1/auth public endpoints.
type AuthHandler struct {
	mode  iamservice.IAMMode
	proxy authProxy
}

// NewAuthHandler builds a handler for the given IAM mode.
func NewAuthHandler(mode iamservice.IAMMode, proxy authProxy) *AuthHandler {
	return &AuthHandler{mode: mode, proxy: proxy}
}

// RegisterAuthRoutes wires /auth routes beneath the API prefix.
func RegisterAuthRoutes(group *gin.RouterGroup, deps *app.Deps) {
	if group == nil || deps == nil {
		return
	}
	handler := NewAuthHandler(deps.IAMMode, deps.AuthProxy)
	authGroup := group.Group("/auth")
	authGroup.Use(middleware.RequestTrace())
	authGroup.POST("/login", handler.Login)
	authGroup.POST("/refresh", handler.Refresh)
	authGroup.POST("/logout", handler.Logout)
	authGroup.GET("/me/context", handler.MeContext)
}

// Login proxies login requests to PowerX Core.
func (h *AuthHandler) Login(c *gin.Context) {
	if !h.ensureAvailable(c) {
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}
	tokens, err := h.proxy.Login(c.Request.Context(), iamservice.LoginRequest{
		Tenant:     req.Tenant,
		Identifier: req.Identifier,
		Password:   req.Password,
		Remember:   req.Remember,
	})
	if err != nil {
		h.handleProxyErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

// Refresh exchanges refresh_token for a new access token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	if !h.ensureAvailable(c) {
		return
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	tokens, err := h.proxy.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleProxyErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

// Logout revokes the current refresh token upstream.
func (h *AuthHandler) Logout(c *gin.Context) {
	if !h.ensureAvailable(c) {
		return
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	if err := h.proxy.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleProxyErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

// MeContext fetches the active user context from PowerX Core.
func (h *AuthHandler) MeContext(c *gin.Context) {
	if !h.ensureAvailable(c) {
		return
	}
	token := extractBearer(c.GetHeader("Authorization"))
	if token == "" {
		contracts.ResponseUnauthorized(c, "缺少 Authorization Bearer token")
		return
	}
	ctx, err := h.proxy.MeContext(c.Request.Context(), token)
	if err != nil {
		h.handleProxyErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, ctx)
}

func (h *AuthHandler) ensureAvailable(c *gin.Context) bool {
	if h.mode != iamservice.IAMModeDelegated {
		contracts.ResponseServiceUnavailable(c, "Local IAM 模式暂未实现", nil)
		return false
	}
	if h.proxy == nil {
		contracts.ResponseServiceUnavailable(c, "宿主认证未配置", nil)
		return false
	}
	return true
}

func (h *AuthHandler) handleProxyErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iamservice.ErrAuthUnavailable):
		contracts.ResponseServiceUnavailable(c, "宿主认证不可用，请稍后重试", nil)
	case errors.Is(err, iamservice.ErrUnauthorized):
		contracts.ResponseUnauthorized(c, "认证失败，请重新登录")
	default:
		var perr *authproxy.ProxyError
		if errors.As(err, &perr) {
			contracts.ResponseError(c, perr.Status, contracts.ErrCodeInternalError, perr.Message)
		} else {
			contracts.ResponseInternalError(c, err)
		}
	}
}

func mapTokens(tokens *iamservice.AuthTokens) gin.H {
	if tokens == nil {
		return gin.H{}
	}
	expiresAt := tokens.ExpiresAt.UTC()
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	return gin.H{
		"token_type":    tokens.TokenType,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
		"expires_at":    expiresAt.UnixMilli(),
		"scope":         tokens.Scope,
	}
}

func extractBearer(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[len("Bearer "):])
	}
	return ""
}

type loginRequest struct {
	Tenant     string `json:"tenant"`
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Remember   bool   `json:"remember"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
