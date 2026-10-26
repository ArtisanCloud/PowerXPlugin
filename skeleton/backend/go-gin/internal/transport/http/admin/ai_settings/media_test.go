package ai_settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMediaRoutesUseBoundRuntimeAndPreserveDiagnostics(t *testing.T) {
	raw, err := os.ReadFile("../../../../../../../../docs/contracts/fixtures/ai-media-v2.json")
	require.NoError(t, err)
	var fixture map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fixture))
	var task powerxai.MediaTask
	require.NoError(t, json.Unmarshal(fixture["task_result"], &task))
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "ApiKey key", r.Header.Get("Authorization"))
		require.Empty(t, r.URL.RawQuery)
		if strings.HasSuffix(r.URL.Path, "/events") {
			require.Equal(t, task.TaskUUID+":2", r.Header.Get("Last-Event-ID"))
			data, err := json.Marshal(task)
			require.NoError(t, err)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "id: %s\nevent: status\ndata: %s\n\n", task.EventID(), data)
			return
		}
		var request struct {
			CapabilityID      string `json:"capability_id"`
			PreferredProtocol string `json:"preferred_protocol"`
			Payload           struct {
				Body map[string]any `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "core_internal", request.PreferredProtocol)
		if request.Payload.Body["model_key"] == "denied" {
			w.Header().Set("X-Trace-ID", "core-trace")
			w.Header().Set("X-Request-ID", "core-request")
			w.WriteHeader(422)
			_, _ = fmt.Fprint(w, `{"error_code":"AI_PARAMETER_INVALID","details":{"reason_code":"AI_PARAMETER_INVALID","parameter":"size","reason":"unsupported_value","upstream_status":400}}`)
			return
		}
		result := fixture["task_result"]
		if request.CapabilityID == powerxai.CapabilityMediaModels {
			result = fixture["catalog_result"]
		}
		_, _ = fmt.Fprintf(w, `{"data":{"protocol_used":"core_internal","payload":{"result":%s}}}`, result)
	}))
	defer core.Close()
	client, err := powerxai.NewClient(powerxai.Config{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "key"}, core.Client())
	require.NoError(t, err)
	runtime, err := fwai.NewRuntime(provider.ModeDelegated, nil, client)
	require.NoError(t, err)
	router := gin.New()
	RegisterRoutes(router.Group("/admin"), &app.Deps{AIInvocation: runtime, ProviderMode: provider.ModeDelegated})
	for _, test := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/admin/ai-settings/media/models?capability=image_generation", "", 200},
		{"POST", "/admin/ai-settings/media/image/invoke", `{"model_key":"qwen-cn/qwen-image-2.0-pro","inputs":[{"type":"text","content":"fixture"}],"idempotency_key":"attempt"}`, 202},
		{"GET", "/admin/ai-settings/media/tasks/" + task.TaskUUID, "", 200},
		{"POST", "/admin/ai-settings/media/tasks/" + task.TaskUUID + "/cancel", "", 200},
		{"POST", "/admin/ai-settings/media/image/invoke", `{"model_key":"denied","inputs":[{"type":"text","content":"fixture"}],"idempotency_key":"attempt"}`, 422},
		{"POST", "/admin/ai-settings/media/image/invoke?env=dev", `{}`, 400},
		{"POST", "/admin/ai-settings/media/image/invoke", `{"endpoint":"/admin/root"}`, 400},
		{"GET", "/admin/ai-settings/media/tasks/" + task.TaskUUID + "/events", "", 200},
	} {
		t.Run(test.method+test.path+fmt.Sprint(test.status), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Last-Event-ID", task.TaskUUID+":2")
			router.ServeHTTP(recorder, req)
			require.Equal(t, test.status, recorder.Code, recorder.Body.String())
			if test.status == 422 {
				require.Equal(t, "core-trace", recorder.Header().Get("X-Trace-ID"))
				require.Equal(t, "core-request", recorder.Header().Get("X-Request-ID"))
				var body map[string]any
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.Equal(t, "size", body["details"].(map[string]any)["parameter"])
			}
			if strings.HasSuffix(test.path, "/events") {
				require.Contains(t, recorder.Body.String(), "id: "+task.EventID())
				require.Contains(t, recorder.Body.String(), "event: status")
			}
		})
	}
}

func TestLocalMediaDoesNotCallCore(t *testing.T) {
	local, err := runtimeexample.NewLocalAI(&config.LocalAIConfig{}, nil)
	require.NoError(t, err)
	coreCalls := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { coreCalls++; w.WriteHeader(500) }))
	defer core.Close()
	client, err := powerxai.NewClient(powerxai.Config{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "key"}, core.Client())
	require.NoError(t, err)
	runtime, err := fwai.NewRuntime(provider.ModeLocal, local, client)
	require.NoError(t, err)
	service, err := runtime.Generative()
	require.NoError(t, err)
	_, err = service.GetMediaTask(context.Background(), "11111111-2222-4333-8444-555555555555")
	require.Error(t, err)
	recorder := httptest.NewRecorder()
	router := gin.New()
	RegisterRoutes(router.Group("/admin"), &app.Deps{AIInvocation: runtime, ProviderMode: provider.ModeLocal})
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/admin/ai-settings/media/models", nil))
	require.Equal(t, 422, recorder.Code)
	require.Zero(t, coreCalls)
}
