package ai_settings

import (
	"context"
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	localrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	localsettings "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/ai_settings"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	fwaisettings "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type aiSettingsInvokerStub struct {
	called bool
	last   gateway.InvokeRequest
}

func TestLocalSettingsHandlerSavesTenantProfile(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalAISetting{}); err != nil {
		t.Fatal(err)
	}
	repo, err := localrepo.NewLocalSettingsRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "fixture", Endpoint: "http://127.0.0.1:11434", Modalities: []string{"llm"}}}}
	ai, err := runtimeexample.NewLocalAI(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := localsettings.NewLocalSettingsService(repo, cfg, ai)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{mode: fwprovider.ModeLocal, deps: &app.Deps{LocalAISettings: svc}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/ai-settings/local-setting", strings.NewReader(`{"source":"local","environment":"development","modality":"llm","provider":"ollama","model_key":"chat","endpoint":"http://127.0.0.1:11434","parameters":{"temperature":0.2}}`)).WithContext(authx.ContextWithTenantUUID(context.Background(), "11111111-1111-4111-8111-111111111111"))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	h.SaveLocalSetting(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
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

func TestLocalSettingsHTTPUsesBoundRuntime(t *testing.T) {
	local, err := runtimeexample.NewLocalAI(&config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "fixture", Endpoint: "http://127.0.0.1:11434", Modalities: []string{"llm"}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{mode: fwprovider.ModeLocal, deps: &app.Deps{AISettings: local}}
	for _, handler := range []gin.HandlerFunc{h.Summary, h.ProviderProfiles, h.ModelProfiles, h.Routing, h.Health, h.Mode} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("GET", "/", nil).WithContext(authx.ContextWithTenantUUID(context.Background(), "11111111-1111-4111-8111-111111111111"))
		handler(c)
		if rec.Code != 200 {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
	}
	d := h.diagnostics()
	if !d.LocalAvailable || d.DelegatedAvailable || !d.ReadOnly {
		t.Fatalf("diagnostics: %+v", d)
	}
}

func TestAISettingsConnectionAndQuickCallUseDelegatedRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/ai/llm/invoke" {
			t.Fatalf("unexpected delegated request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, input)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"output":{"type":"text","text":"OK"}}}`))
	}))
	defer server.Close()

	delegated, err := powerxai.NewClient(powerxai.Config{BaseURL: server.URL, BearerToken: "test-token"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := fwai.NewRuntime(fwprovider.ModeDelegated, nil, delegated)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{mode: fwprovider.ModeDelegated, deps: &app.Deps{AIInvocation: runtime}}
	body := `{"source":"powerx","environment":"development","modality":"llm","provider":"ollama","model_key":"qwen3:8b","endpoint":"http://127.0.0.1:11434","parameters":{"temperature":0.2}}`

	for _, handler := range []gin.HandlerFunc{h.TestConnection, h.QuickCall} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/ai-settings", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		handler(c)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
	if len(requests) != 2 {
		t.Fatalf("delegated invocation count=%d, want 2", len(requests))
	}
	for _, input := range requests {
		if input["model_key"] != "qwen3:8b" {
			t.Fatalf("model_key=%v", input["model_key"])
		}
	}
}
