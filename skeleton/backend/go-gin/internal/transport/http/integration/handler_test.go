package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeCapabilityGateway struct {
	lastParams capgateway.InvokeParams
	result     *capgateway.InvokeResult
	err        error
}

func (f *fakeCapabilityGateway) Enabled() bool { return true }

func (f *fakeCapabilityGateway) Invoke(_ context.Context, params capgateway.InvokeParams) (*capgateway.InvokeResult, error) {
	f.lastParams = params
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeCapabilityGateway) ListPlatformCapabilities(_ context.Context, _ capgateway.ListPlatformCapabilitiesOptions) ([]capgateway.PlatformCapabilityRecord, error) {
	return nil, nil
}

func (f *fakeCapabilityGateway) Close() error { return nil }

func TestInvokeCapabilitySuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fake := &fakeCapabilityGateway{
		result: &capgateway.InvokeResult{
			TraceID: "trace-123",
			Status:  "accepted",
			Data: map[string]any{
				"ok": true,
			},
		},
	}

	handler := &Handler{
		deps: &app.Deps{
			CapabilityGateway: fake,
		},
	}

	body := `{"capabilityId":"com.corex.media.assets.manage","action":"List","payload":{"limit":1}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/capabilities/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PX-Use-Mock", "media")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.InvokeCapability(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, true, resp["success"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "trace-123", data["traceId"])
	require.Equal(t, "List", fake.lastParams.Action)
	require.Equal(t, "media", fake.lastParams.Headers["X-PX-Use-Mock"])
	require.Equal(t, "com.corex.media.assets.manage", fake.lastParams.CapabilityID)
}

func TestInvokeCapabilityUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeCapabilityGateway{
		err: &capgateway.UnavailableError{Reason: "px login required"},
	}
	handler := &Handler{
		deps: &app.Deps{
			CapabilityGateway: fake,
		},
	}

	body := `{"capabilityId":"com.corex.media.assets.manage","action":"List","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/capabilities/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.InvokeCapability(c)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestInvokeCapabilityGatewayError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gwErr := &frameworkgateway.InvocationError{
		TraceID:    "gw-trace",
		StatusCode: http.StatusTooManyRequests,
		Errors: []frameworkgateway.GatewayError{
			{Code: "RATE_LIMIT", Message: "too many"},
		},
	}
	fake := &fakeCapabilityGateway{
		err: gwErr,
	}
	handler := &Handler{
		deps: &app.Deps{
			CapabilityGateway: fake,
		},
	}

	body := `{"capabilityId":"com.corex.media.assets.manage","action":"List","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/capabilities/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.InvokeCapability(c)

	require.Equal(t, http.StatusTooManyRequests, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, false, resp["success"])
	errObj, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "RATE_LIMIT", errObj["code"])
	details, ok := errObj["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gw-trace", details["traceId"])
	upstreamError, ok := details["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "rate_limited", upstreamError["type"])
}

func TestInvokeCapabilityForwardsBearerWithoutTenantHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fake := &fakeCapabilityGateway{
		result: &capgateway.InvokeResult{
			TraceID: "trace-tenant",
			Status:  "accepted",
			Data:    map[string]any{"ok": true},
		},
	}

	handler := &Handler{
		deps: &app.Deps{
			CapabilityGateway: fake,
		},
	}

	body := `{"capabilityId":"com.corex.media.assets.read","action":"List","payload":{"method":"GET","endpoint":"/api/v1/media/assets"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/capabilities/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("tenant_uuid", "aeffc79f-e72a-4fd9-b908-5c150bce3741")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.InvokeCapability(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "Bearer test-token", fake.lastParams.Headers["Authorization"])
	require.NotContains(t, fake.lastParams.Headers, "tenant_uuid")
}
