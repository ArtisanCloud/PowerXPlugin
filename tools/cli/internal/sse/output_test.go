package sse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutput_FilterBySessionAndLevel(t *testing.T) {
	cfg := DefaultOutputConfig()
	cfg.ConsoleOutput = false
	cfg.FileOutput = false
	cfg.FilterBySessionID = "session-ok"
	cfg.MinLevel = "warn"

	output, err := NewOutput(cfg)
	if err != nil {
		t.Fatalf("NewOutput failed: %v", err)
	}
	defer output.Close()

	event := Event{
		Event: "log",
		Data:  `{"message":"skip"}`,
		Fields: map[string]interface{}{
			"sessionId": "other",
			"level":     "info",
		},
	}
	output.WriteEvent(event)

	match := Event{
		Event: "log",
		Data:  `{"message":"keep"}`,
		Fields: map[string]interface{}{
			"sessionId": "session-ok",
			"level":     "error",
		},
	}
	output.WriteEvent(match)

	stats := output.GetStats()
	if stats["filtered_events"].(int64) != 1 {
		t.Fatalf("expected 1 filtered event, got %+v", stats["filtered_events"])
	}
}

func TestOutput_FileWrite(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs.jsonl")

	cfg := DefaultOutputConfig()
	cfg.ConsoleOutput = false
	cfg.FileOutput = true
	cfg.LogFilePath = logPath

	output, err := NewOutput(cfg)
	if err != nil {
		t.Fatalf("NewOutput failed: %v", err)
	}

	event := Event{
		Event: "log",
		Data:  `{"message":"file write"}`,
		Fields: map[string]interface{}{
			"sessionId": "session-1",
			"level":     "info",
		},
	}
	output.WriteEvent(event)
	output.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "file write") {
		t.Fatalf("log file missing entry: %s", string(data))
	}
}
