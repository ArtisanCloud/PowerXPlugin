package ai_settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const onlineTenant = "11111111-1111-4111-8111-111111111111"
const onlineCredential = "22222222-2222-4222-8222-222222222222"

func TestUnsupportedOnlineProviderRetainsDeclaredStaticConfiguration(t *testing.T) {
	svc, _ := onlineService(t, nil)
	in := OnlineCatalogInput{Environment: "development", Modality: "embedding", Provider: "hash"}
	items, err := svc.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	onlineCode(t, err, "AI_MODEL_SYNC_UNSUPPORTED")
	_, err = svc.Save(context.Background(), onlineTenant, "local", SaveInput{Environment: in.Environment, Modality: in.Modality, Provider: in.Provider, ModelKey: items[0]["id"]})
	require.NoError(t, err)
}

func TestLocalProviderCatalogExposesOnlyPublicConnectionDefaults(t *testing.T) {
	svc, _ := onlineService(t, nil)
	p := svc.Catalog.providers["openai"]
	p.Auth.Defaults["api_key"] = "fixture.private_key"
	p.Defaults = map[string]string{"base_url": "https://fixture.invalid/v1", "client_secret": "fixture.private_secret"}
	svc.Catalog.providers["openai"] = p
	providers := svc.CatalogProviders("llm")
	raw, err := json.Marshal(providers)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "fixture.private_key")
	require.NotContains(t, string(raw), "fixture.private_secret")
	for _, provider := range providers {
		if provider["id"] == "openai" {
			require.Equal(t, map[string]string{"base_url": "https://fixture.invalid/v1"}, provider["auth"].(map[string]any)["defaults"])
		}
	}
}

func TestDiscoveredNativeModelIsUsedOnlyByExplicitTrial(t *testing.T) {
	var chatCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"native-private:14b"}]}`))
		case "POST /api/show":
			_, _ = w.Write([]byte(`{"capabilities":["completion"]}`))
		case "POST /api/chat":
			chatCalls.Add(1)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "native-private:14b", body["model"])
			_, _ = w.Write([]byte(`{"message":{"content":"fixture.reply"},"done":true}`))
		default:
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	svc, _ := onlineService(t, nil)
	ctx := authx.ContextWithTenantUUID(context.Background(), onlineTenant)
	in := OnlineCatalogInput{Environment: "development", Modality: "llm", Provider: "ollama", Endpoint: server.URL}
	_, err := svc.SyncOnlineCatalog(ctx, onlineTenant, in, "refresh", "", nil)
	require.NoError(t, err)
	require.Zero(t, chatCalls.Load())
	profile := SaveInput{Environment: in.Environment, Modality: in.Modality, Provider: in.Provider, Endpoint: in.Endpoint, ModelKey: "native-private:14b"}
	_, err = svc.Save(ctx, onlineTenant, "local", profile)
	require.NoError(t, err)
	require.Zero(t, chatCalls.Load())
	result, err := svc.InvokeLLM(ctx, profile, []powerxai.ContentItem{{Type: "text", Content: "fixture.prompt"}})
	require.NoError(t, err)
	require.Equal(t, "fixture.reply", result.Text)
	require.EqualValues(t, 1, chatCalls.Load())
}

func onlineService(t *testing.T, cfg *config.LocalAIConfig) (*LocalSettingsService, *gorm.DB) {
	t.Helper()
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sql.Close()) })
	require.NoError(t, db.AutoMigrate(&models.LocalAISetting{}, &models.LocalAIOnlineCatalog{}))
	repo, err := repository.NewLocalSettingsRepository(db)
	require.NoError(t, err)
	ai, err := runtimeexample.NewLocalAI(nil, nil)
	require.NoError(t, err)
	svc, err := NewLocalSettingsService(repo, cfg, ai)
	require.NoError(t, err)
	svc.Catalog, err = LoadCatalog(filepath.Join("..", "..", "..", "..", "etc", "ai", "providers.d"))
	require.NoError(t, err)
	return svc, db
}
func onlineCode(t *testing.T, err error, code string) {
	t.Helper()
	var typed *powerxai.HTTPError
	require.ErrorAs(t, err, &typed)
	require.Equal(t, code, typed.ReasonCode)
}
func ids(items []map[string]string) []string {
	out := []string{}
	for _, m := range items {
		out = append(out, m["id"])
	}
	return out
}

func TestOnlineOllamaCatalogSavesNativeModelsAndPreservesProfilesOnFailure(t *testing.T) {
	var fail, remove atomic.Bool
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if fail.Load() {
			w.WriteHeader(500)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /api/tags":
			if remove.Load() {
				_, _ = w.Write([]byte(`{"models":[]}`))
			} else {
				_, _ = w.Write([]byte(`{"models":[{"name":"private-chat:14b"},{"name":"private-embed:latest"}]}`))
			}
		case "POST /api/show":
			var in map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			mods := []string{"embedding"}
			if in["model"] == "private-chat:14b" {
				mods = []string{"completion", "vision"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"capabilities": mods})
		default:
			t.Errorf("unexpected endpoint %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	svc, _ := onlineService(t, nil)
	in := OnlineCatalogInput{Environment: "development", Provider: "ollama", Modality: "llm", Endpoint: server.URL}
	result, err := svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	require.NoError(t, err)
	require.Len(t, result.Models, 2)
	items, err := svc.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.Equal(t, []string{"private-chat:14b"}, ids(items))
	saved, err := svc.Save(context.Background(), onlineTenant, "local", SaveInput{Environment: in.Environment, Provider: in.Provider, Modality: in.Modality, ModelKey: "private-chat:14b", Endpoint: in.Endpoint, Parameters: map[string]any{"temperature": 0.25}})
	require.NoError(t, err)
	require.NotEmpty(t, saved.UUID)
	for _, mod := range []string{"embedding", "vlm"} {
		scope := in
		scope.Modality = mod
		items, err = svc.ListLocalCatalog(context.Background(), onlineTenant, scope)
		require.NoError(t, err)
		if mod == "embedding" {
			require.Equal(t, []string{"private-embed:latest"}, ids(items))
		} else {
			require.Equal(t, []string{"private-chat:14b"}, ids(items))
		}
	}
	for _, scope := range []struct{ tenant, env string }{{onlineTenant, "production"}, {"33333333-3333-4333-8333-333333333333", "development"}} {
		other := in
		other.Environment = scope.env
		items, err = svc.ListLocalCatalog(context.Background(), scope.tenant, other)
		require.NoError(t, err)
		require.NotContains(t, ids(items), "private-chat:14b")
	}
	reloaded, err := NewLocalSettingsService(svc.Repository, nil, svc.ai)
	require.NoError(t, err)
	reloaded.Catalog = svc.Catalog
	items, err = reloaded.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.Equal(t, []string{"private-chat:14b"}, ids(items))
	fail.Store(true)
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	onlineCode(t, err, "AI_MODEL_DISCOVERY_UPSTREAM_FAILED")
	row, err := svc.Repository.OnlineCatalog(context.Background(), onlineTenant, in.Environment, in.Provider, in.App)
	require.NoError(t, err)
	require.Equal(t, result.Revision, row.Revision)
	fail.Store(false)
	remove.Store(true)
	empty, err := svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	require.NoError(t, err)
	require.Empty(t, empty.Models)
	profile, err := svc.Get(context.Background(), onlineTenant, in.Environment, in.Modality, "local")
	require.NoError(t, err)
	require.Equal(t, saved.UUID, profile.UUID)
	require.Equal(t, "private-chat:14b", profile.ModelKey)
	require.JSONEq(t, `{"temperature":0.25}`, string(profile.Parameters))
	_, err = svc.Save(context.Background(), onlineTenant, "local", SaveInput{Environment: in.Environment, Provider: in.Provider, Modality: in.Modality, ModelKey: profile.ModelKey, Endpoint: in.Endpoint})
	onlineCode(t, err, "AI_MODEL_NOT_IN_CATALOG")
	require.Equal(t, []string{"GET /api/tags", "POST /api/show", "POST /api/show", "GET /api/tags", "GET /api/tags"}, calls)
}

func TestOnlineCloudCredentialScopeManualClassificationAndKeyRotation(t *testing.T) {
	t.Setenv("FIXTURE_AI_ONLINE_KEY", "fixture.secret")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "GET /v1/models", r.Method+" "+r.URL.Path)
		require.Equal(t, "Bearer fixture.secret", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":[{"id":"private/company:model","output_modalities":["text"]},{"id":"opaque:model"}]}`))
	}))
	defer server.Close()
	cfg := &config.LocalAIConfig{Credentials: []config.LocalAICredential{{Reference: onlineCredential, TenantUUID: onlineTenant, Environment: "development", Provider: "openai", Endpoint: server.URL, APIKeyEnv: "FIXTURE_AI_ONLINE_KEY"}}}
	svc, db := onlineService(t, cfg)
	in := OnlineCatalogInput{Environment: "development", Provider: "openai", Modality: "llm", Endpoint: server.URL, CredentialRef: onlineCredential}
	result, err := svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	require.NoError(t, err)
	items, err := svc.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.Contains(t, ids(items), "private/company:model")
	require.NotContains(t, ids(items), "opaque:model")
	for _, bad := range []OnlineCatalogInput{func() OnlineCatalogInput { x := in; x.Environment = "production"; return x }(), func() OnlineCatalogInput { x := in; x.Endpoint = server.URL + "/other"; return x }(), func() OnlineCatalogInput { x := in; x.Provider = "anthropic"; return x }()} {
		_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, bad, "refresh", "", nil)
		onlineCode(t, err, "AI_CREDENTIAL_SCOPE_MISMATCH")
	}
	_, err = svc.SyncOnlineCatalog(context.Background(), "33333333-3333-4333-8333-333333333333", in, "refresh", "", nil)
	onlineCode(t, err, "AI_CREDENTIAL_SCOPE_MISMATCH")
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "assign", result.Revision, []string{"opaque:model", "missing:model"})
	onlineCode(t, err, "AI_MODEL_SYNC_ASSIGNMENT_INVALID")
	assigned, err := svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "assign", result.Revision, []string{"opaque:model"})
	require.NoError(t, err)
	require.NotEqual(t, result.Revision, assigned.Revision)
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "assign", result.Revision, []string{"opaque:model"})
	onlineCode(t, err, "AI_MODEL_SYNC_STALE")
	items, err = svc.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.Contains(t, ids(items), "opaque:model")
	_, err = svc.Save(context.Background(), onlineTenant, "local", SaveInput{Environment: in.Environment, Provider: in.Provider, Modality: in.Modality, ModelKey: "opaque:model", Endpoint: in.Endpoint, CredentialRef: in.CredentialRef})
	require.NoError(t, err)
	refreshed, err := svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	require.NoError(t, err)
	require.Equal(t, []string{"llm"}, refreshed.Models[1].ManualModalities)
	require.EqualValues(t, 2, calls.Load())
	raw, err := json.Marshal(refreshed)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "fixture.secret")
	require.NotContains(t, string(raw), "fingerprint")
	require.NotContains(t, string(raw), onlineCredential)
	// A failed write preserves both the prior revision and its classifications.
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("fixture.fail_online_write", func(tx *gorm.DB) {
		if tx.Statement.Table == (models.LocalAIOnlineCatalog{}).TableName() {
			tx.AddError(errors.New("fixture.write_failed"))
		}
	}))
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "refresh", "", nil)
	require.Error(t, err)
	require.NoError(t, db.Callback().Update().Remove("fixture.fail_online_write"))
	row, err := svc.Repository.OnlineCatalog(context.Background(), onlineTenant, in.Environment, in.Provider, in.App)
	require.NoError(t, err)
	require.Equal(t, refreshed.Revision, row.Revision)
	t.Setenv("FIXTURE_AI_ONLINE_KEY", "fixture.rotated")
	items, err = svc.ListLocalCatalog(context.Background(), onlineTenant, in)
	require.NoError(t, err)
	require.NotContains(t, ids(items), "private/company:model")
	require.NotContains(t, ids(items), "opaque:model")
	_, err = svc.SyncOnlineCatalog(context.Background(), onlineTenant, in, "assign", refreshed.Revision, []string{"opaque:model"})
	onlineCode(t, err, "AI_MODEL_SYNC_CONNECTION_CHANGED")
}
