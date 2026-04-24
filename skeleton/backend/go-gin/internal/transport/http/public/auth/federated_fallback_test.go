package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	federatedChallenge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/challenge"
	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	federatedProviders "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers"
	federatedRisk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	federatedsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/gin-gonic/gin"
)

type failingProvider struct{}

func (f failingProvider) Key() string { return "wecom" }
func (f failingProvider) BuildAuthorizeURL(context.Context, federatedContracts.AuthorizeRequest) (federatedContracts.AuthorizeResponse, error) {
	return federatedContracts.AuthorizeResponse{AuthorizeURL: "https://example.test/auth"}, nil
}
func (f failingProvider) ExchangeCode(context.Context, federatedContracts.ExchangeCodeRequest) (federatedContracts.ProviderToken, error) {
	return federatedContracts.ProviderToken{}, errors.New("provider down")
}
func (f failingProvider) ResolveIdentity(context.Context, federatedContracts.ResolveIdentityRequest) (federatedContracts.ExternalIdentity, error) {
	return federatedContracts.ExternalIdentity{}, errors.New("unreachable")
}

func TestFederatedFallbackWhenProviderFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := federatedProviders.NewRegistry()
	_ = registry.Register(failingProvider{})
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

	callbackReq := map[string]any{
		"provider":        "wecom",
		"tenant_uuid":     "tenant-a",
		"state":           payload["state"],
		"nonce":           payload["nonce"],
		"code":            "auth-code-fail",
		"signature_valid": true,
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
	if out.Error == nil || out.Error.Message != "登录失败，请稍后重试" {
		t.Fatalf("error=%+v, want generic failure message", out.Error)
	}
}

func TestAuthModeCoexistKeepsPasswordPathEnabled(t *testing.T) {
	svc := federatedsvc.NewAuthModeService("")
	if !svc.FederatedEnabled() || !svc.PasswordEnabled() {
		t.Fatalf("coexist mode expected both enabled")
	}

	_ = os.Setenv("POWERX_FEDERATED_AUTH_MODE", "password_only")
	defer os.Unsetenv("POWERX_FEDERATED_AUTH_MODE")
	onlyPwd := federatedsvc.NewAuthModeService("password_only")
	if onlyPwd.FederatedEnabled() {
		t.Fatalf("password_only should disable federated path")
	}
	if !onlyPwd.PasswordEnabled() {
		t.Fatalf("password_only should keep password enabled")
	}
}
