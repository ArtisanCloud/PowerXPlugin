package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/google/uuid"
)

type tenantContextKey string

const (
	tenantHeaderName                   = "tenant_uuid"
	defaultTenantUUID                  = "00000000-0000-0000-0000-000000000001"
	tenantKey         tenantContextKey = "framework.tenant_uuid"
)

// TenantContext 根据请求头解析租户 ID，缺省时写入默认值。
func TenantContext() bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			if ctx == nil {
				return
			}
			raw := strings.TrimSpace(ctx.Header(tenantHeaderName))
			var tenantUUID string
			if raw == "" {
				tenantUUID = defaultTenantUUID
				ctx.SetHeader(tenantHeaderName, tenantUUID)
			} else {
				if _, err := uuid.Parse(raw); err != nil {
					router.RespondError(ctx, http.StatusBadRequest, "INVALID_TENANT_UUID", "invalid tenant uuid", nil)
					return
				}
				tenantUUID = strings.ToLower(raw)
			}

			current := ctx.Context()
			if current == nil {
				current = context.Background()
			}
			ctx.SetContext(context.WithValue(current, tenantKey, tenantUUID))

			if next != nil {
				next(ctx)
			}
		}
	}
}

// TenantUUIDFromContext 从上下文中读取租户 ID。
func TenantUUIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v, ok := ctx.Value(tenantKey).(string); ok && v != "" {
		return v, true
	}
	return "", false
}

// WithTenantUUID 将租户 ID 写入上下文，可用于测试或内存实现。
func WithTenantUUID(ctx context.Context, tenantUUID string) context.Context {
	if ctx == nil || strings.TrimSpace(tenantUUID) == "" {
		return ctx
	}
	return context.WithValue(ctx, tenantKey, strings.TrimSpace(tenantUUID))
}
