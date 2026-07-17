package taskqueue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type rotatingTokenProvider struct {
	mu          sync.Mutex
	token       string
	tokenCalls  int
	invalidates int
}

func (p *rotatingTokenProvider) Token(context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokenCalls++
	return p.token, nil
}

func (p *rotatingTokenProvider) InvalidateToken() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invalidates++
	p.token = "fresh-token"
}

func TestNewHostProviderRequiresBearerCredential(t *testing.T) {
	_, err := NewHostProvider(HostProviderConfig{
		BaseURL:    "http://127.0.0.1:8077",
		AuthScheme: "bearer",
	})
	if err == nil {
		t.Fatal("expected missing bearer credential error")
	}
	if got := err.Error(); got != "taskqueue host provider: token provider is required for bearer auth" {
		t.Fatalf("error = %q", got)
	}
}

func TestHostProviderRefreshesTokenOnceOnUnauthorized(t *testing.T) {
	provider := &rotatingTokenProvider{token: "expired-token"}
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.Header.Get("Authorization") {
		case "Bearer expired-token":
			http.Error(w, "expired", http.StatusUnauthorized)
		case "Bearer fresh-token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"messages":[]}}`))
		default:
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
	}))
	defer server.Close()

	queue, err := NewHostProvider(HostProviderConfig{
		BaseURL:       server.URL,
		AuthScheme:    "bearer",
		TokenProvider: provider,
	})
	if err != nil {
		t.Fatalf("NewHostProvider() error = %v", err)
	}

	messages, err := queue.Dequeue(context.Background(), DequeueRequest{TenantKey: "tenant", SubscriberID: "sub"})
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("messages len = %d", len(messages))
	}
	if calls != 2 {
		t.Fatalf("http calls = %d", calls)
	}
	if provider.invalidates != 1 {
		t.Fatalf("invalidates = %d", provider.invalidates)
	}
	if provider.tokenCalls != 2 {
		t.Fatalf("token calls = %d", provider.tokenCalls)
	}
}

func TestHostProviderReturnsAuthenticationErrorAfterRetry(t *testing.T) {
	provider := &rotatingTokenProvider{token: "expired-token"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	queue, err := NewHostProvider(HostProviderConfig{
		BaseURL:       server.URL,
		AuthScheme:    "bearer",
		TokenProvider: provider,
	})
	if err != nil {
		t.Fatalf("NewHostProvider() error = %v", err)
	}

	_, err = queue.Dequeue(context.Background(), DequeueRequest{TenantKey: "tenant", SubscriberID: "sub"})
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthenticationError, got %T: %v", err, err)
	}
	if authErr.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", authErr.StatusCode)
	}
	if provider.invalidates != 1 {
		t.Fatalf("invalidates = %d", provider.invalidates)
	}
}
