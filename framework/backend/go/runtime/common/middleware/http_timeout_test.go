package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTimeoutHandlerPreservesCompletedResponse(t *testing.T) {
	w := httptest.NewRecorder()
	TimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.WriteHeader(201)
		_, _ = w.Write([]byte("ok"))
	}), time.Second, nil).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 201 || w.Body.String() != "ok" || len(w.Header().Values("Set-Cookie")) != 2 {
		t.Fatalf("response=%+v", w)
	}
}

func TestTimeoutCancelsAndRejectsLateResponse(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Private", "hidden")
		_, _ = w.Write([]byte("partial"))
		close(started)
		<-r.Context().Done()
		<-release
		w.Header().Set("X-Late", "hidden")
		w.WriteHeader(201)
		_, err := w.Write([]byte("late"))
		finished <- err
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "timeout-test")
	returned := make(chan struct{})
	go func() { TimeoutHandler(app, 50*time.Millisecond, nil).ServeHTTP(w, req); close(returned) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout did not return")
	}
	close(release)
	select {
	case err := <-finished:
		if !errors.Is(err, http.ErrHandlerTimeout) {
			t.Fatalf("late error=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("late handler blocked")
	}
	if w.Code != 408 || w.Header().Get("X-Request-ID") != "timeout-test" || !strings.Contains(w.Body.String(), "REQUEST_TIMEOUT") || strings.Contains(w.Body.String(), "partial") || w.Header().Get("X-Private") != "" || w.Header().Get("X-Late") != "" {
		t.Fatalf("response=%+v", w)
	}
}

func TestTimeoutPropagatesPanic(t *testing.T) {
	defer func() {
		if recover() != "test-panic" {
			t.Error("panic not propagated")
		}
	}()
	TimeoutHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("test-panic") }), time.Second, nil).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestTimeoutRespectsParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done(); close(done) })
	cancel()
	w := httptest.NewRecorder()
	TimeoutHandler(app, time.Hour, nil).ServeHTTP(w, httptest.NewRequest("GET", "/", nil).WithContext(ctx))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler not canceled")
	}
	if w.Body.Len() != 0 {
		t.Fatal("response written after client cancellation")
	}
}
