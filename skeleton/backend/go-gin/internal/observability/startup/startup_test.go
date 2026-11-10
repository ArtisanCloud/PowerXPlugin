package startup

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

func TestReadyNoticeRemainsVisibleWithFileOutputAndErrorLevel(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelError} {
		t.Run(level.String(), func(t *testing.T) {
			file, err := os.Create(filepath.Join(t.TempDir(), "runtime.log"))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			var terminal bytes.Buffer
			started := time.Date(2026, 10, 30, 12, 0, 0, 0, time.FixedZone("test", 8*60*60))
			ready := started.Add(1250 * time.Millisecond)
			r := Reporter{StartedAt: started, Terminal: &terminal, Logger: slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})), PluginID: "fixture.plugin", Mode: "local", Locale: "zh-CN"}
			r.HTTPReady(bootstrap.HTTPReadyInfo{ListenAddr: "[::]:8078", ReadyAt: ready})
			line := terminal.String()
			if !strings.Contains(line, "1.25s") || !strings.Contains(line, "2026-10-30 12:00:01 +08:00") || !strings.Contains(line, "http://127.0.0.1:8078") || strings.Count(line, "\n") != 1 || strings.Contains(line, "%!") {
				t.Fatal("STARTUP_NOTICE_INVALID", line)
			}
			data, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			if level == slog.LevelError {
				if len(data) != 0 {
					t.Fatal("LOG_LEVEL_NOT_PRESERVED")
				}
				return
			}
			var event map[string]any
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatal(err)
			}
			if event["event"] != "plugin.http.ready" || event["startup_duration_ms"] != float64(1250) || event["listen_addr"] != "[::]:8078" || event["ready_at"] != ready.Format(time.RFC3339Nano) || event["started_at"] != started.Format(time.RFC3339Nano) || event["mode"] != "local" || event["plugin_id"] != "fixture.plugin" {
				t.Fatal("STARTUP_EVENT_INVALID", event)
			}
		})
	}
}

func TestReadyNoticeUsesLocaleResources(t *testing.T) {
	for _, locale := range []string{"zh-CN", "en"} {
		t.Run(locale, func(t *testing.T) {
			raw, err := locales.ReadFile("locales/" + locale + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var dictionary map[string]string
			if err := json.Unmarshal(raw, &dictionary); err != nil {
				t.Fatal(err)
			}
			var terminal bytes.Buffer
			at := time.Now()
			Reporter{StartedAt: at, Terminal: &terminal, Locale: locale}.HTTPReady(bootstrap.HTTPReadyInfo{ListenAddr: "127.0.0.1:8078", ReadyAt: at})
			prefix := strings.Split(dictionary["http_ready"], "%s")[0]
			if prefix == "" || !strings.HasPrefix(terminal.String(), prefix) || strings.Contains(terminal.String(), "%!") {
				t.Fatal("STARTUP_LOCALE_INVALID")
			}
		})
	}
}

func TestReadyNoticeRequiresSuccessfulBind(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	var terminal, events bytes.Buffer
	app := bootstrap.NewApp(&bootstrap.Config{Listen: addr, Standalone: true})
	app.Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}
	app.AttachServer(server, server.Shutdown)
	reporter := Reporter{StartedAt: time.Now(), Terminal: &terminal, Logger: slog.New(slog.NewJSONHandler(&events, nil))}
	app.OnHTTPReady(reporter.HTTPReady)
	if err := app.Run(); err == nil {
		t.Fatal("BIND_CONFLICT_NOT_REPORTED")
	}
	if terminal.Len() != 0 || events.Len() != 0 {
		t.Fatal("FAILED_STARTUP_REPORTED_READY")
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan bootstrap.HTTPReadyInfo, 1)
	app.OnHTTPReady(func(info bootstrap.HTTPReadyInfo) { reporter.HTTPReady(info); ready <- info })
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	t.Cleanup(func() { _ = app.Shutdown() })
	select {
	case info := <-ready:
		if info.ListenAddr != addr || terminal.Len() == 0 || events.Len() == 0 {
			t.Fatal("BOUND_STARTUP_NOTICE_MISSING")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP_READY_TIMEOUT")
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatal("HTTP_NOT_SERVING")
	}
	if err := app.Shutdown(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP_DID_NOT_STOP")
	}
	if strings.Count(terminal.String(), "\n") != 1 {
		t.Fatal("DUPLICATE_READY_NOTICE")
	}
	rebound, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal("LISTENER_NOT_RELEASED", err)
	}
	_ = rebound.Close()
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("FIXTURE_WRITE_FAILED") }

func TestTerminalFailureIsRecordedWithoutChangingLogSink(t *testing.T) {
	var events bytes.Buffer
	at := time.Now()
	Reporter{StartedAt: at, Terminal: failingWriter{}, Logger: slog.New(slog.NewJSONHandler(&events, &slog.HandlerOptions{Level: slog.LevelError}))}.HTTPReady(bootstrap.HTTPReadyInfo{ListenAddr: "127.0.0.1:8078", ReadyAt: at})
	var event map[string]any
	if err := json.Unmarshal(events.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event["msg"] != "PLUGIN_STARTUP_TERMINAL_WRITE_FAILED" || event["level"] != "ERROR" {
		t.Fatal("TERMINAL_FAILURE_NOT_RECORDED", event)
	}
}
