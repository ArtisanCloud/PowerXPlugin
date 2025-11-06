package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

type contextKey string

const (
	requestIDHeader            = "X-Request-ID"
	requestIDKey    contextKey = "framework.request_id"
)

// RequestID 确保每个请求拥有请求 ID，并写回上下文/响应头。
func RequestID() bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			if ctx == nil {
				return
			}
			reqID := strings.TrimSpace(ctx.Header(requestIDHeader))
			if reqID == "" {
				reqID = generateRequestID()
			}
			ctx.SetHeader(requestIDHeader, reqID)

			current := ctx.Context()
			if current == nil {
				current = context.Background()
			}
			ctx.SetContext(context.WithValue(current, requestIDKey, reqID))

			if next != nil {
				next(ctx)
			}
		}
	}
}

// RequestIDFromContext 从上下文中读取请求 ID。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "req-" + hex.EncodeToString(b[:8])
}
