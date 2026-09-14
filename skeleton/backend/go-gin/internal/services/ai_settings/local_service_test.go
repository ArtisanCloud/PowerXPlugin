package ai_settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLocalSettingsServicePersistsTenantScopedProfile(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalAISetting{}); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.NewLocalSettingsRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "fixture", Endpoint: "http://127.0.0.1:11434", Modalities: []string{"llm"}}}}
	ai, err := runtimeexample.NewLocalAI(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewLocalSettingsService(repo, cfg, ai)
	if err != nil {
		t.Fatal(err)
	}
	tenant := "11111111-1111-4111-8111-111111111111"
	if _, err := svc.Save(context.Background(), tenant, "local", SaveInput{Environment: "development", Modality: "llm", Provider: "ollama", ModelKey: "chat", Endpoint: "http://127.0.0.1:11434", Parameters: map[string]any{"temperature": 0.2}}); err != nil {
		t.Fatal(err)
	}
	item, err := svc.Get(context.Background(), tenant, "development", "llm", "local")
	if err != nil {
		t.Fatal(err)
	}
	if item == nil || item.UUID == "" || item.Provider != "ollama" || item.ModelKey != "chat" {
		t.Fatalf("saved item=%+v", item)
	}
	other, err := svc.Get(context.Background(), "22222222-2222-4222-8222-222222222222", "development", "llm", "local")
	if err != nil {
		t.Fatal(err)
	}
	if other != nil {
		t.Fatal("profile leaked across tenants")
	}
}

func TestLocalSettingsServiceUsesCatalogSourceOnlyForThisSave(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalAISetting{}); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.NewLocalSettingsRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "fixture", Endpoint: "http://127.0.0.1:11434", Modalities: []string{"llm"}}}}
	ai, err := runtimeexample.NewLocalAI(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewLocalSettingsService(repo, cfg, ai)
	if err != nil {
		t.Fatal(err)
	}
	tenant := "11111111-1111-4111-8111-111111111111"
	// A PowerX-catalog profile is persisted in the plugin, but its catalog
	// source is only a request-time validator and has no durable setting.
	if _, err := svc.Save(context.Background(), tenant, "powerx", SaveInput{Environment: "development", Modality: "llm", Provider: "openai", ModelKey: "gpt-4o"}); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(&models.PluginSystemConfig{}) {
		t.Fatal("catalog source must not create a plugin_system_configs table")
	}
}

func TestLocalSettingsServiceSeparatesLocalAndPowerXProfiles(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalAISetting{}); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.NewLocalSettingsRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	ai, err := runtimeexample.NewLocalAI(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewLocalSettingsService(repo, nil, ai)
	if err != nil {
		t.Fatal(err)
	}
	svc.Catalog, err = LoadCatalog(filepath.Join("..", "..", "..", "..", "etc", "ai", "providers.d"))
	if err != nil {
		t.Fatal(err)
	}
	tenant := "11111111-1111-4111-8111-111111111111"
	if _, err := svc.Save(context.Background(), tenant, "powerx", SaveInput{Environment: "development", Modality: "llm", Provider: "ollama", ModelKey: "qwen3:8b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Save(context.Background(), tenant, "local", SaveInput{Environment: "development", Modality: "llm", Provider: "baidu", ModelKey: "ERNIE-Bot-4", Endpoint: "https://qianfan.baidubce.com/v2"}); err != nil {
		t.Fatal(err)
	}
	local, err := svc.Get(context.Background(), tenant, "development", "llm", "local")
	if err != nil || local == nil || local.Provider != "baidu" || local.ModelKey != "ERNIE-Bot-4" || local.Source != "local" {
		t.Fatalf("local profile=%+v err=%v", local, err)
	}
	powerx, err := svc.Get(context.Background(), tenant, "development", "llm", "powerx")
	if err != nil || powerx == nil || powerx.Provider != "ollama" || powerx.ModelKey != "qwen3:8b" || powerx.Source != "powerx" {
		t.Fatalf("powerx profile=%+v err=%v", powerx, err)
	}
}

func TestLocalSettingsServiceTestsTheProfileInsteadOfStaticRuntimeModels(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := repository.NewLocalSettingsRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path=%s, want /api/chat", r.URL.Path)
		}
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model != "qwen3:8b" {
			t.Fatalf("body=%#v err=%v; want selected profile model", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":{"content":"pong"},"done":true}`))
	}))
	defer server.Close()
	cfg := &config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "fixture", Endpoint: server.URL, Modalities: []string{"llm"}}}}
	ai, err := runtimeexample.NewLocalAI(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewLocalSettingsService(repo, cfg, ai)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.TestConnection(context.Background(), SaveInput{Provider: "ollama", ModelKey: "qwen3:8b", Endpoint: server.URL}); err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
}

func TestPowerXCatalogExposesMultipleProvidersAndModels(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "..", "etc", "ai", "providers.d"))
	if err != nil {
		t.Fatalf("load PowerX AI catalog: %v", err)
	}
	providers := catalog.Providers("llm")
	if len(providers) < 5 {
		t.Fatalf("provider count=%d, want complete PowerX catalog", len(providers))
	}
	if !catalog.HasModel("llm", "openai", "", "gpt-4o") {
		t.Fatal("expected OpenAI gpt-4o from PowerX catalog")
	}
	var ollama map[string]any
	for _, provider := range catalog.Providers("llm") {
		if provider["id"] == "ollama" {
			ollama = provider
			break
		}
	}
	if ollama == nil {
		t.Fatal("expected Ollama provider")
	}
	auth, _ := ollama["auth"].(map[string]any)
	defaults, _ := auth["defaults"].(map[string]string)
	if defaults["base_url"] != "http://127.0.0.1:11434" {
		t.Fatalf("Ollama base_url default=%q", defaults["base_url"])
	}
	ollamaModels := catalog.Models("llm", "ollama", "")
	// qwen-coder is an explicit plugin-local catalog entry. Everything after it
	// remains the copied Core Ollama catalog in Core order.
	wantOllama := []string{"qwen-coder", "llama3", "llama3:70b", "mistral", "mixtral", "qwen2", "qwen3:8b", "deepseek-r1:1.5b", "deepseek-coder", "phi3", "gemma2:9b"}
	if len(ollamaModels) != len(wantOllama) {
		t.Fatalf("Ollama model count=%d, want %d: %#v", len(ollamaModels), len(wantOllama), ollamaModels)
	}
	for i, id := range wantOllama {
		if ollamaModels[i]["id"] != id {
			t.Fatalf("Ollama model[%d]=%q, want %q", i, ollamaModels[i]["id"], id)
		}
	}
}
