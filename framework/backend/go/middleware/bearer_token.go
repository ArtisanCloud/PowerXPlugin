package middleware

import (
	"context"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

type bearerTokenContextKey string

const bearerTokenKey bearerTokenContextKey = "framework.bearer_token"

// BearerToken captures Authorization: Bearer <token> into context for outbound calls.
func BearerToken() bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			if ctx == nil {
				return
			}
			authz := strings.TrimSpace(ctx.Header("Authorization"))
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token := strings.TrimSpace(authz[7:])
				if token != "" {
					current := ctx.Context()
					if current == nil {
						current = context.Background()
					}
					ctx.SetContext(context.WithValue(current, bearerTokenKey, token))
				}
			}
			if next != nil {
				next(ctx)
			}
		}
	}
}

// BearerTokenFromContext returns bearer token stored in context (if any).
func BearerTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(bearerTokenKey).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// WithBearerToken stores bearer token into context.
func WithBearerToken(ctx context.Context, token string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(token) == "" {
		return ctx
	}
	return context.WithValue(ctx, bearerTokenKey, strings.TrimSpace(token))
}
