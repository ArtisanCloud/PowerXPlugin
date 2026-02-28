package integration_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMiniAppDelegateAuth_TenantMismatchForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api/v1")
	api.Use(httpmw.EnsureTenant())

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"success":true,"data":{"tenant_uuid":"00000000-0000-0000-0000-000000000099","customer_uuid":"00000000-0000-0000-0000-000000000002"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &config.Config{
		Server:  &config.ServerConfig{},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:             "delegate",
			DelegateEndpoint: "http://powerx.local/api/v1/customer/auth/validate",
			DelegateTimeout:  "1s",
		},
	}
	auth := customersvc.NewDelegateAuthenticator(cfg, client, nil)
	api.Use(authmw.CustomerAuth(auth, nil))
	api.GET("/mini-app/ping", func(c *gin.Context) {
		contracts.ResponseSuccess(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("X-PowerX-Tenant", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(contracts.ErrCodeTenantMismatch)) {
		t.Fatalf("expected TENANT_MISMATCH, got %s", rec.Body.String())
	}
}

func TestMiniAppDelegateAuth_UpstreamUnavailable503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api/v1")
	api.Use(httpmw.EnsureTenant())

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}
	cfg := &config.Config{
		Server:  &config.ServerConfig{},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:             "delegate",
			DelegateEndpoint: "http://powerx.local/api/v1/customer/auth/validate",
			DelegateTimeout:  "1s",
		},
	}
	auth := customersvc.NewDelegateAuthenticator(cfg, client, nil)
	api.Use(authmw.CustomerAuth(auth, nil))
	api.GET("/mini-app/ping", func(c *gin.Context) {
		contracts.ResponseSuccess(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("X-PowerX-Tenant", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}
