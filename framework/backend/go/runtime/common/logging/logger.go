package logging

import (
	"context"
	"strings"
)

type Fields map[string]any

type Logger interface {
	WithFields(Fields) Logger
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type contextKey string

const runtimeFieldsKey contextKey = "runtime_fields"

func MergeFields(base, extra Fields) Fields {
	out := Fields{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func WithRuntime(ctx context.Context, fields Fields) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	merged := MergeFields(Fields{}, fields)
	return context.WithValue(ctx, runtimeFieldsKey, merged)
}

func RuntimeFieldsFromContext(ctx context.Context) Fields {
	if ctx == nil {
		return Fields{}
	}
	v := ctx.Value(runtimeFieldsKey)
	fields, ok := v.(Fields)
	if !ok || fields == nil {
		return Fields{}
	}
	return MergeFields(Fields{}, fields)
}

func NormalizeRuntimeFields(fields Fields) Fields {
	return ApplyRuntimeFallback(fields)
}

func trimString(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

