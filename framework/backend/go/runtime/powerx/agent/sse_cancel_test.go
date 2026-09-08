package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamCancellationClosesHostRequest(t *testing.T) {
	disconnected := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		fmt.Fprint(w, "event: token\ndata: {\"payload\":{\"text\":\"token\"}}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(disconnected)
	}))
	defer server.Close()
	c, err := NewClient(PowerXAgentClientConfig{BaseURL: server.URL, BearerToken: "test-sts"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count := 0
	err = c.StreamSSE(ctx, nil, func(AgentStreamEvent) error { count++; cancel(); return nil })
	if count != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("events=%d err=%v", count, err)
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("host_request_not_closed")
	}
}

func TestStreamRejectsInvalidClientAndContentType(t *testing.T) {
	var missing *Client
	if err := missing.StreamSSE(context.Background(), nil, func(AgentStreamEvent) error { return nil }); err == nil {
		t.Fatal("nil_client_accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream-invalid")
		fmt.Fprint(w, "event: token\ndata: {}\n\n")
	}))
	defer server.Close()
	c, err := NewClient(PowerXAgentClientConfig{BaseURL: server.URL, BearerToken: "test-sts"})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	err = c.StreamSSE(context.Background(), nil, func(AgentStreamEvent) error { count++; return nil })
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != ErrCodeStreamDecode || count != 0 {
		t.Fatalf("events=%d err=%v", count, err)
	}
}
