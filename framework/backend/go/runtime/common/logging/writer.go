package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type WriterOptions struct {
	Format string
	Level  string
}

type FileOptions struct {
	Path       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
	Cleanup    bool
	Clock      func() time.Time
}

type RegistryOptions struct {
	WriterOptions
	File   FileOptions
	Stdout io.Writer
	Stderr io.Writer
	Loki   Sink
}

type WriterSink struct {
	name   SinkType
	writer io.Writer
	format string
	level  string
}

func NewStdoutSink(opts WriterOptions, writer io.Writer) Sink {
	if writer == nil {
		writer = os.Stdout
	}
	return NewWriterSink(SinkStdout, writer, opts)
}

func NewStderrSink(opts WriterOptions, writer io.Writer) Sink {
	if writer == nil {
		writer = os.Stderr
	}
	return NewWriterSink(SinkType("stderr"), writer, opts)
}

func NewWriterSink(name SinkType, writer io.Writer, opts WriterOptions) Sink {
	return &WriterSink{
		name:   name,
		writer: writer,
		format: normalizeFormat(opts.Format),
		level:  normalizeLevel(opts.Level),
	}
}

func (s *WriterSink) Name() SinkType {
	if s == nil {
		return ""
	}
	return s.name
}

func (s *WriterSink) Emit(_ context.Context, event Event) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("writer sink is not configured")
	}
	if !levelEnabled(s.level, event.Level) {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	record := Fields{
		"time":  event.Timestamp.UTC().Format(time.RFC3339Nano),
		"level": normalizeLevel(event.Level),
		"msg":   strings.TrimSpace(event.Message),
	}
	for k, v := range event.Fields {
		if strings.TrimSpace(k) == "" {
			continue
		}
		record[k] = v
	}
	if record["msg"] == "" {
		record["msg"] = "log event"
	}
	if s.format == "text" {
		_, err := fmt.Fprintf(s.writer, "time=%s level=%s msg=%q fields=%v\n", record["time"], record["level"], record["msg"], event.Fields)
		return err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(s.writer, string(payload))
	return err
}

func NewFileSink(opts FileOptions, writerOpts WriterOptions) (Sink, error) {
	writer, err := NewRotatingFileWriter(opts)
	if err != nil {
		return nil, err
	}
	return NewWriterSink(SinkFile, writer, writerOpts), nil
}

func NewRotatingFileWriter(opts FileOptions) (io.Writer, error) {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if opts.Cleanup {
		if _, err := CleanExpiredLogFiles(path, opts.MaxAgeDays, opts.Clock); err != nil {
			return nil, err
		}
	}
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    positiveOrDefault(opts.MaxSizeMB, 100),
		MaxBackups: positiveOrDefault(opts.MaxBackups, 3),
		MaxAge:     positiveOrDefault(opts.MaxAgeDays, 28),
		Compress:   opts.Compress,
		LocalTime:  true,
	}, nil
}

func BuildDefaultRegistry(opts RegistryOptions) (*SinkRegistry, error) {
	registry := NewSinkRegistry()
	if err := registry.Register(NewStdoutSink(opts.WriterOptions, opts.Stdout)); err != nil {
		return nil, err
	}
	if err := registry.Register(NewStderrSink(opts.WriterOptions, opts.Stderr)); err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.File.Path) != "" {
		fileSink, err := NewFileSink(opts.File, opts.WriterOptions)
		if err != nil {
			return nil, err
		}
		if err := registry.Register(fileSink); err != nil {
			return nil, err
		}
	}
	if opts.Loki != nil {
		if err := registry.Register(opts.Loki); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func NewSlogLogger(writer io.Writer, opts WriterOptions) *slog.Logger {
	if writer == nil {
		writer = os.Stdout
	}
	handlerOpts := &slog.HandlerOptions{Level: slogLevel(opts.Level)}
	if normalizeFormat(opts.Format) == "text" {
		return slog.New(slog.NewTextHandler(writer, handlerOpts))
	}
	return slog.New(slog.NewJSONHandler(writer, handlerOpts))
}

func ResolvePrimaryWriter(policy Policy, file FileOptions, stdout, stderr io.Writer) (io.Writer, error) {
	output := SinkType(PrimaryOutput(policy))
	switch output {
	case SinkFile:
		return NewRotatingFileWriter(file)
	case SinkType("stderr"):
		if stderr == nil {
			return os.Stderr, nil
		}
		return stderr, nil
	default:
		if stdout == nil {
			return os.Stdout, nil
		}
		return stdout, nil
	}
}

func CleanExpiredLogFiles(activePath string, maxAgeDays int, clock func() time.Time) ([]string, error) {
	if maxAgeDays <= 0 {
		return nil, nil
	}
	activePath = strings.TrimSpace(activePath)
	if activePath == "" {
		return nil, nil
	}
	now := time.Now
	if clock != nil {
		now = clock
	}
	dir := filepath.Dir(activePath)
	base := filepath.Base(activePath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	cutoff := now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)
	removed := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == base || !looksLikeRotatedLog(name, stem, ext) {
			continue
		}
		fullPath := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			return removed, err
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(fullPath); err != nil {
			return removed, err
		}
		removed = append(removed, fullPath)
	}
	return removed, nil
}

func looksLikeRotatedLog(name, stem, ext string) bool {
	if ext != "" && !strings.HasSuffix(name, ext) && !strings.HasSuffix(name, ext+".gz") {
		return false
	}
	return strings.HasPrefix(name, stem+"-") || strings.HasPrefix(name, stem+".")
}

func positiveOrDefault(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return "text"
	default:
		return "json"
	}
}

func normalizeLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "warn", "error":
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return "info"
	}
}

func levelEnabled(configured, event string) bool {
	return levelWeight(normalizeLevel(event)) >= levelWeight(normalizeLevel(configured))
}

func levelWeight(level string) int {
	switch normalizeLevel(level) {
	case "debug":
		return 10
	case "warn":
		return 30
	case "error":
		return 40
	default:
		return 20
	}
}

func slogLevel(level string) slog.Leveler {
	switch normalizeLevel(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
