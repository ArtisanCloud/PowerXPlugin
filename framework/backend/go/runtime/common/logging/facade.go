package logging

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Facade struct {
	ctx    context.Context
	logger Logger
	fields Fields
}

func FromContext(ctx context.Context) *Facade {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Facade{
		ctx:    ctx,
		logger: NewSlogAdapter(slog.Default()),
		fields: RuntimeFieldsFromContext(ctx),
	}
}

func (f *Facade) With(extra Fields) *Facade {
	if f == nil {
		return FromContext(nil).With(extra)
	}
	merged := MergeFields(f.fields, extra)
	return &Facade{ctx: f.ctx, logger: f.logger, fields: merged}
}

func (f *Facade) Emit(level, message string, extra Fields) {
	if f == nil {
		FromContext(nil).Emit(level, message, extra)
		return
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "log event"
	}
	fields := NormalizeRuntimeFields(MergeFields(f.fields, extra))
	fields["event_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry := f.logger.WithFields(fields)
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		entry.Error(msg)
	case "warn", "warning":
		entry.Warn(msg)
	default:
		entry.Info(msg)
	}
}
