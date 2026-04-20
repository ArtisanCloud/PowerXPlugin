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
	providerWeCom "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/wecom"
	federatedRisk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/gin-gonic/gin"
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
