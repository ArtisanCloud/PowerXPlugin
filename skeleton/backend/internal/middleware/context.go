package middleware

// internal/middleware/context.go

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

type TenantContext struct {
	TenantID      int64    `json:"tenant_id"`
	UserID        int64    `json:"user_id"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	PolicyVersion string   `json:"policy_version"`
}

const (
	ctxKeyTenant = "tenant_ctx"
	ctxKeyToken  = "raw_bearer_token"
)

type tenantIDContextKey struct{}

var ctxKeyTenantID = tenantIDContextKey{}

var ErrTenantMissing = errors.New("tenant context missing")

func SetTenantContext(c *gin.Context, tc TenantContext) { c.Set(ctxKeyTenant, tc) }
func GetTenantContext(c *gin.Context) (TenantContext, bool) {
	v, ok := c.Get(ctxKeyTenant)
	if !ok || v == nil {
		return TenantContext{}, false
	}
	tc, ok := v.(TenantContext)
	return tc, ok
}
func SetRawBearerToken(c *gin.Context, token string) {
	if token != "" {
		c.Set(ctxKeyToken, token)
	}
}
func GetRawBearerToken(c *gin.Context) (string, bool) {
	v, ok := c.Get(ctxKeyToken)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

// ContextWithTenantID stores tenant ID into a standard context.
func ContextWithTenantID(ctx context.Context, tenantID uint64) context.Context {
	if ctx == nil || tenantID == 0 {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyTenantID, tenantID)
}

// TenantIDFromContext extracts tenant ID from a standard context.
func TenantIDFromContext(ctx context.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	if v := ctx.Value(ctxKeyTenantID); v != nil {
		switch id := v.(type) {
		case uint64:
			if id > 0 {
				return id, true
			}
		case int64:
			if id > 0 {
				return uint64(id), true
			}
		case int:
			if id > 0 {
				return uint64(id), true
			}
		}
	}
	return 0, false
}

// RequireTenantID retrieves tenant ID from context or returns ErrTenantMissing.
func RequireTenantID(ctx context.Context) (uint64, error) {
	if tenantID, ok := TenantIDFromContext(ctx); ok && tenantID > 0 {
		return tenantID, nil
	}
	return 0, ErrTenantMissing
}
