package middleware

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// AuthGuardOptions 描述统一授权判定中间件输入。
type AuthGuardOptions struct {
	Authz          contracts.AuthzService
	Resource       string
	Action         string
	TenantResolver func(bootstrap.Context) string
	UserResolver   func(bootstrap.Context) string
	TraceResolver  func(bootstrap.Context) string
}

// AuthGuard 提供统一 IAM 授权判定输出。
func AuthGuard() bootstrap.Middleware {
	return AuthGuardWithOptions(AuthGuardOptions{})
}

// AuthGuardWithOptions 可注入资源动作与上下文解析规则。
func AuthGuardWithOptions(opts AuthGuardOptions) bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			if ctx == nil {
				return
			}
			if opts.Authz == nil {
				router.RespondError(ctx, http.StatusFailedDependency, iamerrors.CodeUpstreamDependency, "authz service is unavailable", nil)
				return
			}

			tenantUUID := resolveTenantUUID(ctx, opts.TenantResolver)
			if tenantUUID == "" {
				router.RespondError(ctx, http.StatusUnauthorized, iamerrors.CodeUnauthorized, "tenant context missing", nil)
				return
			}

			decision, err := opts.Authz.Authorize(ctx.Context(), contracts.AuthorizationRequest{
				TenantUUID: tenantUUID,
				UserID:     resolveUserID(ctx, opts.UserResolver),
				Resource:   strings.TrimSpace(opts.Resource),
				Action:     strings.TrimSpace(opts.Action),
				TraceID:    resolveTraceID(ctx, opts.TraceResolver),
			})
			if err != nil {
				respondAuthzError(ctx, err)
				return
			}
			if decision != nil && !decision.Allowed {
				router.RespondError(ctx, http.StatusForbidden, iamerrors.CodeForbidden, "permission denied", map[string]any{
					"resource": decision.Resource,
					"action":   decision.Action,
					"reason":   decision.ReasonCode,
				})
				return
			}
			if decision == nil {
				router.RespondError(ctx, http.StatusFailedDependency, iamerrors.CodeUpstreamDependency, "empty authz decision", nil)
				return
			}
			if next != nil {
				next(ctx)
			}
		}
	}
}

func resolveTenantUUID(ctx bootstrap.Context, resolver func(bootstrap.Context) string) string {
	if resolver != nil {
		if tenant := strings.TrimSpace(resolver(ctx)); tenant != "" {
			return tenant
		}
	}
	if tenant, ok := TenantUUIDFromContext(ctx.Context()); ok {
		return strings.TrimSpace(tenant)
	}
	return ""
}

func resolveUserID(ctx bootstrap.Context, resolver func(bootstrap.Context) string) string {
	if resolver == nil {
		return ""
	}
	return strings.TrimSpace(resolver(ctx))
}

func resolveTraceID(ctx bootstrap.Context, resolver func(bootstrap.Context) string) string {
	if resolver != nil {
		if traceID := strings.TrimSpace(resolver(ctx)); traceID != "" {
			return traceID
		}
	}
	return strings.TrimSpace(RequestIDFromContext(ctx.Context()))
}

func respondAuthzError(ctx bootstrap.Context, err error) {
	status := iamerrors.StatusCode(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	code := iamerrors.CodeOf(err)
	if code == "" {
		code = iamerrors.CodeUpstreamDependency
	}
	router.RespondError(ctx, status, code, err.Error(), nil)
}
