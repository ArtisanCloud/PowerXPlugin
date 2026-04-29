package logger

import "context"

// WithComponent attaches component field onto an entry.
func WithComponent(component string) *Entry {
	return WithFields(Fields{
		"component": component,
	})
}

func InfoWith(entry *Entry, ctx context.Context, message string, fields map[string]interface{}) {
	if entry == nil {
		InfoCtx(WithLogFields(ctx, fields), message)
		return
	}
	entry.WithContext(ctx).WithFields(Fields(fields)).Info(message)
}

func WarnWith(entry *Entry, ctx context.Context, message string, fields map[string]interface{}) {
	if entry == nil {
		WarnCtx(WithLogFields(ctx, fields), message)
		return
	}
	entry.WithContext(ctx).WithFields(Fields(fields)).Warn(message)
}

func ErrorWith(entry *Entry, ctx context.Context, message string, fields map[string]interface{}) {
	if entry == nil {
		ErrorCtx(WithLogFields(ctx, fields), message)
		return
	}
	entry.WithContext(ctx).WithFields(Fields(fields)).Error(message)
}

func DebugWith(entry *Entry, ctx context.Context, message string, fields map[string]interface{}) {
	if entry == nil {
		DebugCtx(WithLogFields(ctx, fields), message)
		return
	}
	entry.WithContext(ctx).WithFields(Fields(fields)).Debug(message)
}

func FatalWith(entry *Entry, ctx context.Context, message string, fields map[string]interface{}) {
	if entry == nil {
		WithFields(Fields(fields)).WithContext(ctx).Fatal(message)
		return
	}
	entry.WithContext(ctx).WithFields(Fields(fields)).Fatal(message)
}
