package ai_settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fwaisettings "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingsCatalogInvoker struct {
	requests []gateway.InvokeRequest
	err      error
	payload  any
}

func (s *settingsCatalogInvoker) Invoke(_ context.Context, request gateway.InvokeRequest) (*gateway.Response, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return nil, s.err
	}
	return &gateway.Response{Data: map[string]any{"protocol_used": "core_internal", "payload": map[string]any{"result": s.payload}}}, nil
}

func TestPowerXLLMCatalogUsesSettingsClientWithoutGenerativeAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the generative API must not supply an AI Settings catalog")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()
	generative, err := powerxai.NewClient(powerxai.Config{BaseURL: server.URL, BearerToken: "fixture.token"}, server.Client())
	require.NoError(t, err)
	invoker := &settingsCatalogInvoker{}
	settings, err := fwaisettings.NewClient(fwaisettings.Config{Invoker: invoker})
	require.NoError(t, err)
	h := &Handler{deps: &app.Deps{AI: generative, PowerXAISettings: settings}}
	for _, providers := range []bool{true, false} {
		invoker.payload = map[string]any{"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "providers": []any{map[string]any{"id": "ollama", "name": "fixture.core-provider", "apps": []any{}}}}
		path := "/ai-settings/catalog/providers?source=powerx&modality=llm"
		handler := h.CatalogProviders
		operation := "list_settings_providers"
		if !providers {
			invoker.payload = map[string]any{"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "provider": "ollama", "app": "", "items": []any{map[string]string{"model_key": "ollama/instance:tag", "model": "instance:tag", "provider": "ollama", "label": "instance:tag"}}}
			path = "/ai-settings/catalog/models?source=powerx&modality=llm&provider=ollama"
			handler = h.CatalogModels
			operation = "list_settings_models"
		}
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, path, nil)
		handler(c)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		var response struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, "ai_settings", response.Data["catalog_kind"])
		request := invoker.requests[len(invoker.requests)-1]
		require.Equal(t, "core_internal", request.PreferredProtocol)
		require.Equal(t, operation, request.Payload.(map[string]any)["body"].(map[string]any)["operation"])
	}
}

func TestPowerXLLMCatalogFailureNeverFallsBackToOtherDirectories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &settingsCatalogInvoker{err: &gateway.InvocationError{StatusCode: 400, TraceID: "fixture.trace", Body: []byte(`{"error_code":"CAPABILITY_INVOKE_FAILED","details":{"reason_code":"AI_OPERATION_INVALID"}}`)}}
	settings, err := fwaisettings.NewClient(fwaisettings.Config{Invoker: invoker})
	require.NoError(t, err)
	for _, client := range []*fwaisettings.Client{nil, settings} {
		h := &Handler{deps: &app.Deps{PowerXAISettings: client}}
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/ai-settings/catalog/providers?source=powerx&modality=llm", nil)
		h.CatalogProviders(c)
		if client == nil {
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		} else {
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "AI_OPERATION_INVALID")
			require.Equal(t, "fixture.trace", recorder.Header().Get("X-Trace-ID"))
		}
		require.NotContains(t, recorder.Body.String(), `"items"`)
	}
}
