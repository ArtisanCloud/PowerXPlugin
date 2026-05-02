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
	requestIDHeader              = "X-Request-ID"
	traceIDHeader                = "X-Trace-Id"
	traceparentHeader            = "traceparent"
	requestIDKey      contextKey = "framework.request_id"
	traceIDKey        contextKey = "framework.trace_id"
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
			traceID := resolveTraceID(ctx, reqID)
			ctx.SetHeader(requestIDHeader, reqID)
			ctx.SetHeader(traceIDHeader, traceID)

			current := ctx.Context()
			if current == nil {
				current = context.Background()
			}
			current = context.WithValue(current, requestIDKey, reqID)
			current = context.WithValue(current, traceIDKey, traceID)
			ctx.SetContext(current)

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

// TraceIDFromContext 从上下文中读取 trace ID。
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

func resolveTraceID(ctx bootstrap.Context, requestID string) string {
	if ctx == nil {
		return requestID
	}
	if traceparent := strings.TrimSpace(ctx.Header(traceparentHeader)); traceparent != "" {
		if traceID := extractTraceIDFromTraceparent(traceparent); traceID != "" {
			return traceID
		}
	}
	if traceID := strings.TrimSpace(ctx.Header(traceIDHeader)); traceID != "" {
		return traceID
	}
	return requestID
}

func extractTraceIDFromTraceparent(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 {
		return ""
	}
	traceID := strings.TrimSpace(parts[1])
	if len(traceID) != 32 {
		return ""
	}
	return traceID
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "req-" + hex.EncodeToString(b[:8])
}
