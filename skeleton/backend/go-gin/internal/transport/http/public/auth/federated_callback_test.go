package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	federatedChallenge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/challenge"
	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	federatedProviders "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers"
	providerLark "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/lark"
	providerWeCom "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/wecom"
	federatedRisk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	federatedsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestFederatedCallbackFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := federatedProviders.NewRegistry()
	if err := registry.Register(providerWeCom.New("corp-test")); err != nil {
		t.Fatalf("register provider err=%v", err)
	}
	pluginbootstrap.SetFederatedForTests(&pluginbootstrap.FederatedRuntime{
		Factory:   registry,
		Challenge: federatedChallenge.NewManager(),
		Risk:      federatedRisk.NewEvaluator(0),
	})
	t.Cleanup(func() { pluginbootstrap.SetFederatedForTests(nil) })

	r := gin.New()
	group := r.Group("/api/v1")
	RegisterRoutes(group, nil)

	challengeBody := `{"provider":"wecom","tenant_uuid":"tenant-a","redirect_uri":"https://plugin.test/callback"}`
	creq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/challenge", bytes.NewBufferString(challengeBody))
	creq.Header.Set("Content-Type", "application/json")
	cres := httptest.NewRecorder()
	r.ServeHTTP(cres, creq)
	if cres.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", cres.Code, cres.Body.String())
	}

	var challengeResp contracts.APIResponse
	if err := json.Unmarshal(cres.Body.Bytes(), &challengeResp); err != nil {
		t.Fatalf("unmarshal challenge resp err=%v", err)
	}
	payload := challengeResp.Data.(map[string]any)
	state := payload["state"].(string)
	nonce := payload["nonce"].(string)

	callbackReq := map[string]any{
		"provider":        "wecom",
		"tenant_uuid":     "tenant-a",
		"state":           state,
		"nonce":           nonce,
		"code":            "auth-code-1",
		"signature_valid": true,
	}
	buf, _ := json.Marshal(callbackReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/callback", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", res.Code, res.Body.String())
	}

	var callbackResp contracts.APIResponse
	if err := json.Unmarshal(res.Body.Bytes(), &callbackResp); err != nil {
		t.Fatalf("unmarshal callback resp err=%v", err)
	}
	respData := callbackResp.Data.(map[string]any)
	ctxMap := respData["context"].(map[string]any)
	if ctxMap["provider"] != "wecom" {
		t.Fatalf("context provider=%v, want wecom", ctxMap["provider"])
	}
	if ctxMap["tenant_uuid"] != "tenant-a" {
		t.Fatalf("context tenant_uuid=%v, want tenant-a", ctxMap["tenant_uuid"])
	}
}

func TestBuildBrowserCallbackRedirectURI_MultiProvider(t *testing.T) {
	front := "http://127.0.0.1:3131/users/login?redirect=%2Fintro"
	providers := []string{"wecom", "dingtalk", "lark"}
	for _, provider := range providers {
		got, ok := buildBrowserCallbackRedirectURI(front, "https://debug.artisan-cloud.com", provider, "tenant-a", "s-1", "n-1")
		if !ok {
			t.Fatalf("provider=%s expect rewrite success", provider)
		}
		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("provider=%s parse err=%v", provider, err)
		}
		if parsed.Scheme != "https" || parsed.Host != "debug.artisan-cloud.com" {
			t.Fatalf("provider=%s host mismatch: %s", provider, parsed.String())
		}
		q := parsed.Query()
		if q.Get("provider") != provider {
			t.Fatalf("provider=%s query provider mismatch: %s", provider, q.Get("provider"))
		}
		if q.Get("tenant_uuid") != "tenant-a" || q.Get("state") != "s-1" || q.Get("nonce") != "n-1" {
			t.Fatalf("provider=%s query core params mismatch: %v", provider, q)
		}
		if q.Get("front_redirect_uri") != front {
			t.Fatalf("provider=%s front_redirect_uri mismatch: %s", provider, q.Get("front_redirect_uri"))
		}
	}
}

func TestBuildBrowserCallbackRedirectURI_InvalidHost(t *testing.T) {
	got, ok := buildBrowserCallbackRedirectURI("http://127.0.0.1:3131/users/login", "https://", "wecom", "tenant-a", "s-1", "n-1")
	if ok {
		t.Fatalf("expect rewrite failed for invalid host, got=%s", got)
	}
}

func TestFederatedCallbackRiskRejectReturnsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := federatedProviders.NewRegistry()
	if err := registry.Register(providerWeCom.New("corp-test")); err != nil {
		t.Fatalf("register provider err=%v", err)
	}
	pluginbootstrap.SetFederatedForTests(&pluginbootstrap.FederatedRuntime{
		Factory:   registry,
		Challenge: federatedChallenge.NewManager(),
		Risk:      federatedRisk.NewEvaluator(0),
	})
	t.Cleanup(func() { pluginbootstrap.SetFederatedForTests(nil) })

	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), nil)

	challengeBody := `{"provider":"wecom","tenant_uuid":"tenant-a","redirect_uri":"https://plugin.test/callback"}`
	creq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/challenge", bytes.NewBufferString(challengeBody))
	creq.Header.Set("Content-Type", "application/json")
	cres := httptest.NewRecorder()
	r.ServeHTTP(cres, creq)

	var challengeResp contracts.APIResponse
	_ = json.Unmarshal(cres.Body.Bytes(), &challengeResp)
	payload := challengeResp.Data.(map[string]any)

	invalid := false
	callbackReq := map[string]any{
		"provider":        "wecom",
		"tenant_uuid":     "tenant-a",
		"state":           payload["state"],
		"nonce":           payload["nonce"],
		"code":            "auth-code-1",
		"signature_valid": invalid,
	}
	buf, _ := json.Marshal(callbackReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/callback", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	var out contracts.APIResponse
	_ = json.Unmarshal(res.Body.Bytes(), &out)
	if out.Error == nil || out.Error.Code != string(federatedContracts.ErrorCodeRiskSignature) {
		t.Fatalf("error=%+v, want risk signature code", out.Error)
	}
}

func TestLarkChallengeCallbackE2E_WithTenantKeyAndCallbackHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const tenantUUID = "00000000-0000-0000-0000-000000000001"
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:lark-e2e?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE integration_secrets (
id TEXT PRIMARY KEY,
tenant_uuid TEXT NOT NULL,
integration_type TEXT NOT NULL,
current_secret_ref TEXT,
pending_secret_ref TEXT,
rotation_interval_days INTEGER NOT NULL DEFAULT 30,
last_rotated_at DATETIME,
next_rotation_due_at DATETIME,
status TEXT NOT NULL DEFAULT 'ACTIVE',
audit_log JSON,
metadata JSON,
created_at DATETIME,
updated_at DATETIME
)`).Error; err != nil {
		t.Fatalf("create integration_secrets failed: %v", err)
	}
	if err := db.Create(&model.SecretCredential{
		ID:               "secret-lark-e2e",
		TenantUuid:       tenantUUID,
		IntegrationType:  federatedsvc.IntegrationTypeIAMFederatedLark,
		RotationInterval: 30,
		Status:           model.SecretStatusActive,
		Metadata: datatypes.JSONMap{
			"tenant_key":    "tenant_key_001",
			"app_id":        "cli_lark_app_001",
			"app_secret":    "secret_lark_001",
			"callback_host": "https://debug.artisan-cloud.com",
		},
	}).Error; err != nil {
		t.Fatalf("seed lark config failed: %v", err)
	}

	registry := federatedProviders.NewRegistry()
	if err := registry.Register(providerLark.NewWithConfig(providerLark.Config{
		AppID:       "cli_lark_app_001",
		AppSecret:   "secret_lark_001",
		TenantKey:   "tenant_key_001",
		CallbackURL: "https://debug.artisan-cloud.com",
	})); err != nil {
		t.Fatalf("register lark provider err=%v", err)
	}
	pluginbootstrap.SetFederatedForTests(&pluginbootstrap.FederatedRuntime{
		Factory:   registry,
		Challenge: federatedChallenge.NewManager(),
		Risk:      federatedRisk.NewEvaluator(0),
	})
	t.Cleanup(func() { pluginbootstrap.SetFederatedForTests(nil) })

	deps := &app.Deps{DB: db}
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), deps)

	challengeReq := map[string]any{
		"provider":     "lark",
		"tenant_uuid":  tenantUUID,
		"redirect_uri": "http://127.0.0.1:3131/users/login?redirect=%2Fintro",
	}
	challengeBody, _ := json.Marshal(challengeReq)
	chReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/challenge", bytes.NewBuffer(challengeBody))
	chReq.Header.Set("Content-Type", "application/json")
	chRes := httptest.NewRecorder()
	r.ServeHTTP(chRes, chReq)
	if chRes.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", chRes.Code, chRes.Body.String())
	}

	var chResp contracts.APIResponse
	if err := json.Unmarshal(chRes.Body.Bytes(), &chResp); err != nil {
		t.Fatalf("unmarshal challenge resp err=%v", err)
	}
	chData := chResp.Data.(map[string]any)
	authorizeURL := chData["authorize_url"].(string)
	parsedAuthURL, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize_url err=%v", err)
	}
	redirectURI, _ := url.QueryUnescape(parsedAuthURL.Query().Get("redirect_uri"))
	if redirectURI == "" {
		t.Fatalf("redirect_uri missing in authorize_url")
	}
	if gotHost := mustParseURLHost(t, redirectURI); gotHost != "debug.artisan-cloud.com" {
		t.Fatalf("redirect_uri host mismatch: %s", gotHost)
	}
	redirectParsed, _ := url.Parse(redirectURI)
	if redirectParsed.Query().Get("provider") != "lark" {
		t.Fatalf("provider mismatch in rewritten redirect_uri: %s", redirectURI)
	}
	if redirectParsed.Query().Get("tenant_uuid") != tenantUUID {
		t.Fatalf("tenant_uuid mismatch in rewritten redirect_uri: %s", redirectURI)
	}

	callbackReq := map[string]any{
		"provider":        "lark",
		"tenant_uuid":     tenantUUID,
		"state":           chData["state"],
		"nonce":           chData["nonce"],
		"code":            "lark-auth-code-001",
		"signature_valid": true,
	}
	cbBody, _ := json.Marshal(callbackReq)
	cbReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/federated/callback", bytes.NewBuffer(cbBody))
	cbReq.Header.Set("Content-Type", "application/json")
	cbRes := httptest.NewRecorder()
	r.ServeHTTP(cbRes, cbReq)
	if cbRes.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", cbRes.Code, cbRes.Body.String())
	}
	var cbResp contracts.APIResponse
	if err := json.Unmarshal(cbRes.Body.Bytes(), &cbResp); err != nil {
		t.Fatalf("unmarshal callback resp err=%v", err)
	}
	cbData := cbResp.Data.(map[string]any)
	ctxMap := cbData["context"].(map[string]any)
	if ctxMap["provider"] != "lark" {
		t.Fatalf("context provider=%v, want lark", ctxMap["provider"])
	}
	if ctxMap["tenant_uuid"] != tenantUUID {
		t.Fatalf("context tenant_uuid=%v, want %s", ctxMap["tenant_uuid"], tenantUUID)
	}
}

func mustParseURLHost(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url err=%v raw=%s", err, raw)
	}
	return parsed.Hostname()
}
