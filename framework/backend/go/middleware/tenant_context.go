package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

			current := ctx.Context()
			if current == nil {
				current = context.Background()
			}

			candidates := []tenantCandidate{
				{source: "context", value: tenantFromStdContext(current)},
				{source: "token", value: tenantFromAuthorizationHeader(ctx.Header("Authorization"))},
				{source: "header", value: strings.TrimSpace(ctx.Header(tenantHeaderName))},
				{source: "query", value: strings.TrimSpace(ctx.Query(tenantHeaderName))},
			}

			var tenantUUID string
			for _, candidate := range candidates {
				if candidate.value == "" {
					continue
				}
				normalized, err := normalizeTenantUUID(candidate.value)
				if err != nil {
					router.RespondError(ctx, http.StatusBadRequest, "INVALID_TENANT_UUID", "invalid tenant uuid", map[string]any{"source": candidate.source})
					return
				}
				tenantUUID = normalized
				break
			}

			if tenantUUID == "" {
				tenantUUID = defaultTenantUUID
			}

			ctx.SetHeader(tenantHeaderName, tenantUUID)
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

type tenantCandidate struct {
	source string
	value  string
}

func tenantFromStdContext(ctx context.Context) string {
	if tid, ok := TenantUUIDFromContext(ctx); ok {
		return strings.TrimSpace(tid)
	}
	return ""
}

func tenantFromAuthorizationHeader(header string) string {
	header = strings.TrimSpace(header)
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	token := strings.TrimSpace(header[7:])
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if v, ok := claims["tid"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			return s
		}
	}
	if v, ok := claims["tenant_uuid"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			return s
		}
	}
	return ""
}

func normalizeTenantUUID(raw string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return strings.ToLower(parsed.String()), nil
}
