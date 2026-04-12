package unit_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDelegateAuthenticator_SuccessAndCache(t *testing.T) {
	callCount := 0
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		callCount++
		body := `{"success":true,"data":{"tenant_uuid":"00000000-0000-0000-0000-000000000001","customer_uuid":"00000000-0000-0000-0000-000000000002","roles":["buyer"],"exp":1005}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	})

	cfg := &config.Config{
		Server:  &config.ServerConfig{},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:             "delegate",
			DelegateEndpoint: "http://powerx.local/api/v1/customer/auth/validate",
			DelegateTimeout:  "1s",
			CacheTTLSeconds:  60,
		},
	}
	client := &http.Client{Transport: transport}
	auth := customersvc.NewDelegateAuthenticator(cfg, client, nil)
	auth.NowForTest(func() time.Time { return time.Unix(1000, 0) })

	cc, err := auth.Authenticate(context.Background(), "00000000-0000-0000-0000-000000000001", "token-1")
	if err != nil {
		t.Fatalf("Authenticate err=%v", err)
	}
	if cc == nil || !cc.Authenticated {
		t.Fatalf("expected authenticated context, got %#v", cc)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 upstream call, got %d", callCount)
	}

	cc, err = auth.Authenticate(context.Background(), "00000000-0000-0000-0000-000000000001", "token-1")
	if err != nil {
		t.Fatalf("Authenticate(2) err=%v", err)
	}
	if cc == nil || !cc.Authenticated {
		t.Fatalf("expected authenticated context, got %#v", cc)
	}
	if callCount != 1 {
		t.Fatalf("expected cache hit, got %d calls", callCount)
	}
}

func TestDelegateAuthenticator_UpstreamUnauthorized(t *testing.T) {
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewBufferString(`{"success":false}`)),
			Header:     make(http.Header),
		}, nil
	})
	cfg := &config.Config{
		Server:  &config.ServerConfig{},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:             "delegate",
			DelegateEndpoint: "http://powerx.local/api/v1/customer/auth/validate",
			DelegateTimeout:  "1s",
		},
	}
	auth := customersvc.NewDelegateAuthenticator(cfg, &http.Client{Transport: transport}, nil)
	_, err := auth.Authenticate(context.Background(), "00000000-0000-0000-0000-000000000001", "bad-token")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != customersvc.ErrCustomerTokenInvalid {
		t.Fatalf("expected ErrCustomerTokenInvalid, got %v", err)
	}
}
