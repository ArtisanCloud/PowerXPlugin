package ai_settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	local "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/ai_settings"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type probeInvoker struct {
	calls   int
	request gateway.InvokeRequest
}

func (p *probeInvoker) Invoke(_ context.Context, in gateway.InvokeRequest) (*gateway.Response, error) {
	p.calls++
	p.request = in
	return &gateway.Response{Data: map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"result": fw.ConnectionTestResult{ContractVersion: 1, OK: true, Environment: "dev", Modality: "llm", Verification: fw.ConnectionVerification{Provider: "ollama", Probe: "ollama_tags", EndpointReachable: true, Authentication: "not_required"}}}}}, nil
}
func probeRequest(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai-settings/test-connection", strings.NewReader(body)).WithContext(authx.ContextWithTenantUUID(context.Background(), "11111111-1111-4111-8111-111111111111"))
	c.Request.Header.Set("Content-Type", "application/json")
	h.TestConnection(c)
	return rec
}
func TestConnectionPowerXSourceDoesNotDependOnBusinessRuntimeOrModel(t *testing.T) {
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		t.Run(string(mode), func(t *testing.T) {
			i := &probeInvoker{}
			client, err := fw.NewClient(fw.Config{Invoker: i})
			require.NoError(t, err)
			h := &Handler{mode: mode, deps: &app.Deps{PowerXAISettings: client}}
			rec := probeRequest(t, h, `{"source":"powerx","modality":"llm","provider":"ollama"}`)
			require.Equal(t, 200, rec.Code, rec.Body.String())
			require.Equal(t, 1, i.calls)
			require.Equal(t, fw.CapabilityAIConnectionTest, i.request.CapabilityID)
			raw, err := json.Marshal(i.request.Payload)
			require.NoError(t, err)
			require.JSONEq(t, `{"body":{"contract_version":1,"operation":"test_connection","modality":"llm","provider":"ollama"}}`, string(raw))
			require.Contains(t, rec.Body.String(), `"generation_submitted":false`)
			require.Contains(t, rec.Body.String(), `"model_access_verified":false`)
		})
	}
}
func TestConnectionRejectsGenerationAndConnectionOverrides(t *testing.T) {
	for _, extra := range []string{`"prompt":"fixture"`, `"inputs":[]`, `"parameters":{}`, `"idempotency_key":"attempt"`, `"api_key":"fixture"`, `"endpoint":"https://fixture.invalid"`, `"environment":"production"`, `"tenant_uuid":"11111111-1111-4111-8111-111111111111"`} {
		t.Run(extra, func(t *testing.T) {
			i := &probeInvoker{}
			client, err := fw.NewClient(fw.Config{Invoker: i})
			require.NoError(t, err)
			h := &Handler{mode: provider.ModeLocal, deps: &app.Deps{PowerXAISettings: client}}
			rec := probeRequest(t, h, `{"source":"powerx","modality":"llm","provider":"ollama",`+extra+`}`)
			require.Equal(t, 400, rec.Code)
			require.Zero(t, i.calls)
		})
	}
}
func TestConnectionLocalSourceNeverInvokesCoreOrChat(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/tags", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	repo, err := repository.NewLocalSettingsRepository(db)
	require.NoError(t, err)
	ai, err := runtimeexample.NewLocalAI(&config.LocalAIConfig{}, nil)
	require.NoError(t, err)
	service, err := local.NewLocalSettingsService(repo, nil, ai)
	require.NoError(t, err)
	i := &probeInvoker{}
	core, err := fw.NewClient(fw.Config{Invoker: i})
	require.NoError(t, err)
	h := &Handler{mode: provider.ModeLocal, deps: &app.Deps{LocalAISettings: service, PowerXAISettings: core}}
	input := `{"source":"local","environment":"development","modality":"image","provider":"ollama","endpoint":"` + server.URL + `"}`
	rec := probeRequest(t, h, input)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, 1, calls)
	require.Zero(t, i.calls)
	var out struct{ Data fw.ConnectionTestResult }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, "development", out.Data.Environment)
	require.False(t, out.Data.Verification.APIKeyVerified)
	require.False(t, out.Data.Verification.GenerationSubmitted)
	rec = probeRequest(t, h, strings.ReplaceAll(input, `"provider":"ollama"`, `"provider":"openai"`))
	require.Equal(t, 501, rec.Code)
	require.Equal(t, 1, calls)
	require.Zero(t, i.calls)
	rec = httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/mode", nil)
	h.Mode(ctx)
	require.Contains(t, rec.Body.String(), `"connection_probes":{"local":true,"powerx":true}`)
}
func TestConnectionUnavailableClientDoesNotSwitchAdapters(t *testing.T) {
	h := &Handler{mode: provider.ModeLocal, deps: &app.Deps{}}
	rec := probeRequest(t, h, `{"source":"powerx","modality":"llm","provider":"ollama"}`)
	require.Equal(t, 503, rec.Code)
}
