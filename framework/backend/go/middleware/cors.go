package middleware

import (
	"net/http"
	"strings"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

const (
	defaultAllowedHeaders = "Content-Type, Content-Length, Accept-Encoding, " +
		"X-CSRF-Token, Authorization, accept, origin, Cache-Control, " +
		"X-Requested-With, X-PowerX-CTX, X-PowerX-CTX-SIG, X-PowerX-CTX-JWT"
	defaultAllowedMethods = "GET, POST, PUT, DELETE, PATCH, OPTIONS"
)

// CORS returns a middleware that applies permissive CORS headers for development use.
// In production scenarios the caller should provide explicit allow-list options.
func CORS(allowedOrigins ...string) bootstrap.Middleware {
	originValidator := func(origin string) string {
		if origin == "" {
			return ""
		}
		if len(allowedOrigins) == 0 {
			return origin
		}
		for _, allowed := range allowedOrigins {
			if allowed == "*" {
				return origin
			}
			if strings.EqualFold(strings.TrimSpace(allowed), origin) {
				return origin
			}
		}
		return ""
	}

	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			requestOrigin := ctx.Header("Origin")
			if allowed := originValidator(requestOrigin); allowed != "" {
				ctx.SetHeader("Access-Control-Allow-Origin", allowed)
				ctx.SetHeader("Vary", "Origin")
			}
			ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			ctx.SetHeader("Access-Control-Allow-Headers", defaultAllowedHeaders)
			ctx.SetHeader("Access-Control-Allow-Methods", defaultAllowedMethods)
			ctx.SetHeader("Access-Control-Max-Age", "600")

			if strings.EqualFold(ctx.Method(), http.MethodOptions) {
				ctx.Status(http.StatusNoContent)
				return
			}

			if next != nil {
				next(ctx)
			}
		}
	}
}
