package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Facade struct {
	ctx    context.Context
	logger Logger
	fields Fields
}

type Entry struct {
	Fields  Fields
	Context Fields
}

func FromContext(ctx context.Context) *Facade {
	return NewFacade(ctx, nil)
}

func NewFacade(ctx context.Context, log Logger) *Facade {
	if ctx == nil {
		ctx = context.Background()
	}
	if log == nil {
		log = NewSlogAdapter(slog.Default())
	}
	return &Facade{
		ctx:    ctx,
		logger: log,
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
	f.log(level, message, Entry{Fields: extra})
}

func (f *Facade) Info(message string, entry Entry) {
	f.log("info", message, entry)
}

func (f *Facade) Warn(message string, entry Entry) {
	f.log("warn", message, entry)
}

func (f *Facade) Error(message string, entry Entry) {
	f.log("error", message, entry)
}

func (f *Facade) log(level, message string, entry Entry) {
	if f == nil {
		FromContext(nil).log(level, message, entry)
		return
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "log event"
	}
	fields := NormalizeContextFields(f.fields, entry.Fields)
	contextFields := sanitizeContext(entry.Context)
	labels := buildLabels(level, fields, contextFields)
	fields["labels"] = labels
	for k, v := range labels {
		if _, exists := fields[k]; !exists {
			fields[k] = v
		}
	}
	fields["event_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	loggerEntry := f.logger.WithFields(fields)
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		loggerEntry.Error(msg)
	case "warn", "warning":
		loggerEntry.Warn(msg)
	default:
		loggerEntry.Info(msg)
	}
}

var labelContextWhitelist = map[string]struct{}{
	"module": {},
}

var highCardinalityLabelDenylist = map[string]struct{}{
	"tenant_uuid": {},
	"session_id":  {},
	"message_id":  {},
	"trace_id":    {},
	"request_id":  {},
	"user_id":     {},
}

func sanitizeContext(ctx Fields) Fields {
	if len(ctx) == 0 {
		return Fields{}
	}
	out := Fields{}
	for k, v := range ctx {
		key := strings.TrimSpace(strings.ToLower(k))
		if key == "" {
			continue
		}
		if _, ok := labelContextWhitelist[key]; !ok {
			continue
		}
		if _, blocked := highCardinalityLabelDenylist[key]; blocked {
			continue
		}
		if s := trimString(v); s != "" {
			out[key] = s
		}
	}
	return out
}

func buildLabels(level string, fields, context Fields) Fields {
	labels := fixedLabels()
	for k, v := range context {
		labels[k] = v
	}
	normalizedLevel := strings.ToLower(strings.TrimSpace(level))
	if normalizedLevel == "" {
		normalizedLevel = "info"
	}
	labels["level"] = normalizedLevel
	if module := trimString(fields[FieldComponent]); module != "" {
		if _, exists := labels["module"]; !exists {
			labels["module"] = module
		}
	}
	for key := range highCardinalityLabelDenylist {
		delete(labels, key)
	}
	return labels
}

func fixedLabels() Fields {
	return Fields{
		"system":   firstNonEmpty(os.Getenv("POWERX_LOG_SYSTEM"), "powerx"),
		"service":  firstNonEmpty(os.Getenv("POWERX_LOG_SERVICE"), "backend"),
		"env":      firstNonEmpty(os.Getenv("POWERX_ENV"), os.Getenv("POWERX_SERVER_MODE"), "dev"),
		"instance": firstNonEmpty(os.Getenv("HOSTNAME"), "local"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
