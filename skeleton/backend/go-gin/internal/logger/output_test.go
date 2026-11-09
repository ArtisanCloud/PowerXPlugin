package logger

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func captureConsole(t *testing.T) func() (string, string) {
	t.Helper()
	stdout, stderr := os.Stdout, os.Stderr
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = ow, ew
	return func() (string, string) {
		os.Stdout, os.Stderr = stdout, stderr
		_ = ow.Close()
		_ = ew.Close()
		o, _ := io.ReadAll(or)
		e, _ := io.ReadAll(er)
		_ = or.Close()
		_ = er.Close()
		return string(o), string(e)
	}
}

func TestFileLoggingKeepsAllBridgesOffConsoleAndFatalVisible(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	path := filepath.Join(t.TempDir(), "runtime.log")
	finish := captureConsole(t)
	if err := Init("debug", "json", "file", path, 1, 2, 28, true); err != nil {
		finish()
		t.Fatal(err)
	}
	Info("FIXTURE_LOGRUS_BRIDGE")
	Warn("FIXTURE_WARNING")
	Error("FIXTURE_ERROR")
	slog.Debug("FIXTURE_SLOG_BRIDGE")
	log.Print("FIXTURE_STDLIB_BRIDGE")
	logrus.WithField("component", "fixture").Info("FIXTURE_STANDARD_LOGRUS_BRIDGE")
	entry := logrus.NewEntry(Logger)
	entry.Level, entry.Message, entry.Data = logrus.FatalLevel, "FIXTURE_FATAL", logrus.Fields{"error": errors.New("FIXTURE_FATAL_CAUSE")}
	if err := Logger.Hooks.Fire(logrus.FatalLevel, entry); err != nil {
		finish()
		t.Fatal(err)
	}
	stdout, stderr := finish()
	if stdout != "" {
		t.Fatal("FILE_LOGGING_WROTE_STDOUT", stdout)
	}
	var fatal map[string]any
	if json.Unmarshal([]byte(stderr), &fatal) != nil || fatal["event"] != "plugin.fatal" || fatal["error"] != "FIXTURE_FATAL_CAUSE" {
		t.Fatal("FATAL_NOT_VISIBLE_OR_ROUTINE_LOGS_LEAKED", stderr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"FIXTURE_LOGRUS_BRIDGE", "FIXTURE_WARNING", "FIXTURE_ERROR", "FIXTURE_SLOG_BRIDGE", "FIXTURE_STDLIB_BRIDGE", "FIXTURE_STANDARD_LOGRUS_BRIDGE", "FIXTURE_FATAL"} {
		if !strings.Contains(string(data), key) {
			t.Fatal("LOG_BRIDGE_DID_NOT_REACH_FILE", key)
		}
	}
}

func TestInvalidFileOutputFailsWithoutConsoleFallback(t *testing.T) {
	finish := captureConsole(t)
	err := Init("info", "json", "file", "", 1, 1, 1, true)
	out, e := finish()
	if err == nil || out != "" || e != "" {
		t.Fatal("FILE_INIT_SILENTLY_FELL_BACK", err, out, e)
	}
}
