package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

type tenantContextKey string

const (
	tenantHeaderName                  = "X-Tenant-ID"
	defaultTenantID  uint64           = 1
	tenantKey        tenantContextKey = "framework.tenant_id"
)

// TenantContext 根据请求头解析租户 ID，缺省时写入默认值。
func TenantContext() bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			if ctx == nil {
				return
			}
			raw := strings.TrimSpace(ctx.Header(tenantHeaderName))
			var tenantID uint64
			if raw == "" {
				tenantID = defaultTenantID
				ctx.SetHeader(tenantHeaderName, strconv.FormatUint(tenantID, 10))
			} else {
				id, err := strconv.ParseUint(raw, 10, 64)
				if err != nil || id == 0 {
					router.RespondError(ctx, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant id", nil)
					return
				}
				tenantID = id
			}

			current := ctx.Context()
			if current == nil {
				current = context.Background()
			}
			ctx.SetContext(context.WithValue(current, tenantKey, tenantID))

			if next != nil {
				next(ctx)
			}
		}
	}
}

// TenantIDFromContext 从上下文中读取租户 ID。
func TenantIDFromContext(ctx context.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	if v, ok := ctx.Value(tenantKey).(uint64); ok && v > 0 {
		return v, true
	}
	return 0, false
}

// WithTenantID 将租户 ID 写入上下文，可用于测试或内存实现。
func WithTenantID(ctx context.Context, tenantID uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, tenantKey, tenantID)
}
