package logging

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterSinkEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	sink := NewStdoutSink(WriterOptions{Format: "json", Level: "info"}, &buf)

	err := sink.Emit(context.Background(), Event{
		Message: "framework logger event",
		Level:   "info",
		Fields: Fields{
			FieldTraceID: "trace-001",
		},
	})
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"msg":"framework logger event"`) {
		t.Fatalf("expected msg in output, got %s", out)
	}
	if !strings.Contains(out, `"trace_id":"trace-001"`) {
		t.Fatalf("expected trace_id in output, got %s", out)
	}
}

func TestNewFileSinkCreatesParentAndWrites(t *testing.T) {
	base := t.TempDir()
	logPath := filepath.Join(base, "logs", "runtime.log")
	sink, err := NewFileSink(FileOptions{
		Path:       logPath,
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
		Cleanup:    true,
	}, WriterOptions{Format: "json", Level: "info"})
	if err != nil {
		t.Fatalf("new file sink failed: %v", err)
	}
	if err := sink.Emit(context.Background(), Event{Message: "file sink event", Level: "info"}); err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if !strings.Contains(string(data), "file sink event") {
		t.Fatalf("expected file sink output, got %s", string(data))
	}
}

func TestCleanExpiredLogFilesRemovesRotatedFilesOnly(t *testing.T) {
	base := t.TempDir()
	active := filepath.Join(base, "runtime.log")
	oldRotated := filepath.Join(base, "runtime-2026-01-01T00-00-00.000.log")
	newRotated := filepath.Join(base, "runtime-2026-06-27T00-00-00.000.log")
	other := filepath.Join(base, "other-2026-01-01.log")
	for _, path := range []string{active, oldRotated, newRotated, other} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	oldTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldRotated, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}
	if err := os.Chtimes(newRotated, newTime, newTime); err != nil {
		t.Fatalf("chtimes new: %v", err)
	}
	if err := os.Chtimes(other, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes other: %v", err)
	}

	removed, err := CleanExpiredLogFiles(active, 30, func() time.Time {
		return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if len(removed) != 1 || removed[0] != oldRotated {
		t.Fatalf("expected only old rotated file removed, got %#v", removed)
	}
	if _, err := os.Stat(oldRotated); !os.IsNotExist(err) {
		t.Fatalf("old rotated file should be removed, stat err=%v", err)
	}
	for _, path := range []string{active, newRotated, other} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}
