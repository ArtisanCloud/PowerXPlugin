// Package startup reports HTTP readiness independently of the configured log sink.
package startup

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

//go:embed locales/*.json
var locales embed.FS

// Reporter must be attached to App.OnHTTPReady, after routes and dependencies
// are registered. HTTP readiness does not certify background workers or gRPC.
type Reporter struct {
	StartedAt time.Time
	Terminal  io.Writer
	Logger    *slog.Logger
	Locale    string
	PluginID  string
	Mode      string
}

func (r Reporter) HTTPReady(info bootstrap.HTTPReadyInfo) {
	duration := info.ReadyAt.Sub(r.StartedAt)
	attrs := []slog.Attr{
		slog.String("module", "startup"),
		slog.String("event", "plugin.http.ready"),
		slog.String("plugin_id", r.PluginID),
		slog.String("mode", r.Mode),
		slog.String("started_at", r.StartedAt.Format(time.RFC3339Nano)),
		slog.String("ready_at", info.ReadyAt.Format(time.RFC3339Nano)),
		slog.Int64("startup_duration_ms", duration.Milliseconds()),
		slog.String("listen_addr", info.ListenAddr),
		slog.String("url", browserURL(info.ListenAddr)),
	}
	if r.Logger != nil {
		r.Logger.LogAttrs(context.Background(), slog.LevelInfo, "PLUGIN_HTTP_READY", attrs...)
	}
	if r.Terminal == nil {
		return
	}
	locale := "zh-CN"
	if strings.HasPrefix(strings.ToLower(r.Locale), "en") {
		locale = "en"
	}
	raw, err := locales.ReadFile("locales/" + locale + ".json")
	var messages struct {
		HTTPReady string `json:"http_ready"`
	}
	if err == nil {
		err = json.Unmarshal(raw, &messages)
	}
	if err == nil && messages.HTTPReady == "" {
		err = fmt.Errorf("STARTUP_LOCALE_MESSAGE_REQUIRED")
	}
	if err == nil {
		_, err = fmt.Fprintf(r.Terminal, messages.HTTPReady,
			info.ReadyAt.Format("2006-01-02 15:04:05 -07:00"), duration.Round(time.Millisecond), browserURL(info.ListenAddr))
	}
	if err != nil && r.Logger != nil {
		r.Logger.LogAttrs(context.Background(), slog.LevelError, "PLUGIN_STARTUP_TERMINAL_WRITE_FAILED", append(attrs, slog.Any("error", err))...)
	}
}

func browserURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}).String()
}
