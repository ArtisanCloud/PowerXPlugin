package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func readyTestApp(addr string) *App {
	app := NewApp(&Config{Listen: addr, Standalone: true})
	app.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	server := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}
	app.AttachServer(server, server.Shutdown)
	return app
}

func TestHTTPReadyAfterListenerBoundAndRequestsCanBeServed(t *testing.T) {
	app := readyTestApp("127.0.0.1:0")
	ready := make(chan HTTPReadyInfo, 2)
	started := time.Now()
	app.OnHTTPReady(func(info HTTPReadyInfo) { ready <- info })
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	t.Cleanup(func() { _ = app.Shutdown() })
	var info HTTPReadyInfo
	select {
	case info = <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP_READY_NOT_NOTIFIED")
	}
	if info.ReadyAt.Before(started) || strings.HasSuffix(info.ListenAddr, ":0") {
		t.Fatal("READY_INFO_INVALID")
	}
	// At notification time the port is owned by this App, not merely configured.
	if listener, err := net.Listen("tcp", info.ListenAddr); err == nil {
		_ = listener.Close()
		t.Fatal("READY_BEFORE_BIND")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + info.ListenAddr)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
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
	if len(ready) != 0 {
		t.Fatal("DUPLICATE_READY_NOTIFICATION")
	}
	listener, err := net.Listen("tcp", info.ListenAddr)
	if err != nil {
		t.Fatal("LISTENER_NOT_RELEASED", err)
	}
	_ = listener.Close()
}

func TestHTTPBindFailureDoesNotNotifyReady(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	app := readyTestApp(listener.Addr().String())
	called := false
	app.OnHTTPReady(func(HTTPReadyInfo) { called = true })
	if err := app.Run(); err == nil {
		t.Fatal("BIND_CONFLICT_NOT_REPORTED")
	}
	if called {
		t.Fatal("FAILED_STARTUP_REPORTED_READY")
	}
}

func TestShutdownBeforeRunDoesNotBindOrNotifyReady(t *testing.T) {
	app := readyTestApp("127.0.0.1:0")
	called := false
	app.OnHTTPReady(func(HTTPReadyInfo) { called = true })
	if err := app.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("STOPPED_APP_REPORTED_READY")
	}
}

func TestRunWithoutReadyHandlerStillServes(t *testing.T) {
	app := readyTestApp("127.0.0.1:0")
	// Obtain a concrete unoccupied port; bind races must be explicit failures.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	app.server.Addr = addr
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	t.Cleanup(func() { _ = app.Shutdown() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			break
		}
		select {
		case err := <-done:
			t.Fatal("HTTP_RUN_FAILED", err)
		case <-ctx.Done():
			t.Fatal("HTTP_NOT_STARTED")
		case <-time.After(5 * time.Millisecond):
		}
	}
	_ = app.Shutdown()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("HTTP_DID_NOT_STOP")
	}
}
