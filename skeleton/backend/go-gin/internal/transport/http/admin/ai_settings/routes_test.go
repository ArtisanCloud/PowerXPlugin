package ai_settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fwaisettings "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type aiSettingsInvokerStub struct {
	called bool
	last   gateway.InvokeRequest
}

func (s *aiSettingsInvokerStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.called = true
	s.last = req
	return &gateway.Response{
		TraceID: "trace-ai-settings",
		Status:  "ok",
		Data: map[string]any{
			"payload": map[string]any{"status": "healthy"},
		},
	}, nil
}

func TestAISettingsHandlerDelegatedSummaryUsesFrameworkClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &aiSettingsInvokerStub{}
	client, err := fwaisettings.NewClient(fwaisettings.Config{Invoker: invoker})
	if err != nil {
		t.Fatalf("ai settings client: %v", err)
	}
	h := &Handler{mode: fwprovider.ModeDelegated, deps: &app.Deps{AISettings: client}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai-settings/summary", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.Summary(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !invoker.called {
		t.Fatal("expected delegated AI settings handler to call framework gateway invoker")
	}
	if invoker.last.CapabilityID != fwaisettings.CapabilityAISettingsAdminRead {
		t.Fatalf("capability=%s", invoker.last.CapabilityID)
	}
}

func TestAISettingsHandlerLocalMissingProviderReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{mode: fwprovider.ModeLocal}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai-settings/summary", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.Summary(c)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
