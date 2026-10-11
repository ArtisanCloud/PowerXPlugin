package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TimeoutHandler must wrap the entire application handler, not run c.Next in
// a goroutine. Only the application goroutine owns its framework context.
// Streaming requests must be explicitly bypassed before response buffering.
func TimeoutHandler(next http.Handler, timeout time.Duration, bypass func(*http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bypass != nil && bypass(r) {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		req := r.Clone(ctx)
		requestID := req.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
			req.Header.Set("X-Request-ID", requestID)
		}
		buffer := &timeoutResponse{header: make(http.Header)}
		// Buffered completion cannot leave a late handler blocked on send.
		done := make(chan any, 1)
		go func() {
			defer func() { done <- recover() }()
			next.ServeHTTP(buffer, req)
		}()
		select {
		case failure := <-done:
			if failure != nil {
				panic(failure)
			}
			if ctx.Err() == nil {
				buffer.mu.Lock()
				defer buffer.mu.Unlock()
				for key, values := range buffer.header {
					w.Header()[key] = append([]string(nil), values...)
				}
				status := buffer.status
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write(buffer.body.Bytes())
				return
			}
		case <-ctx.Done():
		}
		buffer.mu.Lock()
		buffer.expired = true
		buffer.body.Reset()
		buffer.mu.Unlock()
		if ctx.Err() == context.Canceled {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusRequestTimeout)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": requestID,
			"timestamp":  time.Now().UTC(),
			"error":      map[string]string{"code": "REQUEST_TIMEOUT", "message": "REQUEST_TIMEOUT"},
		})
	})
}

// header is owned only by the application until done; timeout paths never
// read it. Body and expiration share a mutex so late writes fail deterministically.
type timeoutResponse struct {
	mu      sync.Mutex
	header  http.Header
	body    bytes.Buffer
	status  int
	expired bool
}

func (w *timeoutResponse) Header() http.Header { return w.header }
func (w *timeoutResponse) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.expired && w.status == 0 {
		w.status = status
	}
}
func (w *timeoutResponse) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.expired {
		return 0, http.ErrHandlerTimeout
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}
