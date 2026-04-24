package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
)

func TestInitWithFileOutputCreatesParentDirAndWrites(t *testing.T) {
	base := t.TempDir()
	logPath := filepath.Join(base, "nested", "runtime.log")

	Init("info", "json", "file", logPath, 1, 1, 1, true)
	Info("file-output-test")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if !strings.Contains(string(data), "file-output-test") {
		t.Fatalf("expected log content in file, got %s", string(data))
	}
}

func TestInitWithFileOutputFallbackToStdoutWhenPathEmpty(t *testing.T) {
	Init("info", "json", "file", "", 1, 1, 1, true)

	buf := &bytes.Buffer{}
	SetOutput(buf)
	Info("stdout-fallback-test")

	if !strings.Contains(buf.String(), "stdout-fallback-test") {
		t.Fatalf("expected fallback message in stdout buffer, got %s", buf.String())
	}
}

func TestHostModePolicyForcesStdoutJSON(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	p := runtimelogging.ResolveWithHostDefaults(runtimelogging.Policy{
		Mode:   runtimelogging.ModeHost,
		Sinks:  []runtimelogging.SinkType{runtimelogging.SinkFile},
		Format: "text",
		Level:  "info",
		Retry: runtimelogging.RetryPolicy{
			Enabled:     true,
			MaxAttempts: 1,
			BackoffMS:   1,
		},
		AuthorizedExtraSinks: []runtimelogging.SinkType{runtimelogging.SinkFile},
	})
	if got := runtimelogging.PrimaryOutput(p); got != "stdout" {
		t.Fatalf("expected stdout output in host mode, got %s", got)
	}
	if p.Format != "json" {
		t.Fatalf("expected json format in host mode, got %s", p.Format)
	}
}
