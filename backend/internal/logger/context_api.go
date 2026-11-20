package logger

import (
	"context"
	"strings"
)

type logFieldsKey struct{}

// WithLogFields appends structured fields into context for subsequent logger.Info/Warn/Error calls.
func WithLogFields(ctx context.Context, fields map[string]interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	merged := map[string]interface{}{}
	if existing, ok := ctx.Value(logFieldsKey{}).(map[string]interface{}); ok && existing != nil {
		for k, v := range existing {
			merged[k] = v
		}
	}
	for k, v := range fields {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		merged[key] = v
	}
	return context.WithValue(ctx, logFieldsKey{}, merged)
}

func fieldsFromContext(ctx context.Context) Fields {
	if ctx == nil {
		return nil
	}
	raw, ok := ctx.Value(logFieldsKey{}).(map[string]interface{})
	if !ok || raw == nil {
		return nil
	}
	out := Fields{}
	for k, v := range raw {
		out[k] = v
	}
	return out
}

func InfoCtx(ctx context.Context, message string) {
	WithFields(fieldsFromContext(ctx)).WithContext(ctx).Info(message)
}

func WarnCtx(ctx context.Context, message string) {
	WithFields(fieldsFromContext(ctx)).WithContext(ctx).Warn(message)
}

func ErrorCtx(ctx context.Context, message string) {
	WithFields(fieldsFromContext(ctx)).WithContext(ctx).Error(message)
}

func DebugCtx(ctx context.Context, message string) {
	WithFields(fieldsFromContext(ctx)).WithContext(ctx).Debug(message)
}
