package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitSetsDefaultLoggerAndUsesStdoutWriter(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	t.Cleanup(func() { slog.SetDefault(orig) })

	runtime, err := Init(Config{
		Policy: Policy{
			Mode:   ModeStandalone,
			Sinks:  []SinkType{SinkStdout},
			Format: "json",
			Level:  "info",
			Retry:  RetryPolicy{Enabled: true, MaxAttempts: 1, BackoffMS: 1},
		},
		Stdout:     &buf,
		SetDefault: true,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if runtime == nil || runtime.Logger == nil {
		t.Fatal("expected runtime logger")
	}

	slog.Info("framework init event")
	if !strings.Contains(buf.String(), "framework init event") {
		t.Fatalf("expected default logger to write to buffer, got %s", buf.String())
	}
}

func TestInitFileOutputWritesThroughFrameworkRuntime(t *testing.T) {
	base := t.TempDir()
	logPath := filepath.Join(base, "logs", "runtime.log")
	runtime, err := Init(Config{
		Policy: Policy{
			Mode:   ModeStandalone,
			Sinks:  []SinkType{SinkFile},
			Format: "json",
			Level:  "info",
			Retry:  RetryPolicy{Enabled: true, MaxAttempts: 1, BackoffMS: 1},
		},
		File: FileOptions{
			Path:       logPath,
			MaxSizeMB:  1,
			MaxBackups: 1,
			MaxAgeDays: 1,
			Cleanup:    true,
		},
		CleanupFiles: true,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	runtime.Logger.InfoContext(context.Background(), "file init event")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if !strings.Contains(string(data), "file init event") {
		t.Fatalf("expected file log output, got %s", string(data))
	}
}
