package dingtalk

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

// Provider 是钉钉默认 provider 实现（可被插件覆写配置）。
type Provider struct {
	appKey string
}

func New(appKey string) *Provider {
	return &Provider{appKey: strings.TrimSpace(appKey)}
}

func (p *Provider) Key() string { return "dingtalk" }

func (p *Provider) BuildAuthorizeURL(_ context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	q := url.Values{}
	q.Set("client_id", p.appKey)
	q.Set("redirect_uri", req.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", firstNonEmpty(req.Scope, "openid"))
	q.Set("state", req.State)
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	return contracts.AuthorizeResponse{AuthorizeURL: "https://login.dingtalk.com/oauth2/auth?" + q.Encode()}, nil
}

func (p *Provider) ExchangeCode(_ context.Context, req contracts.ExchangeCodeRequest) (contracts.ProviderToken, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "code is required")
	}
	return contracts.ProviderToken{
		AccessToken: "dingtalk-token-" + code,
		Raw: map[string]any{
			"code":    code,
			"app_key": strings.TrimSpace(p.appKey),
		},
	}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("dingtalk:%s", token)
	tenantScope := strings.TrimSpace(p.appKey)
	if req.Token.Raw != nil {
		tenantScope = firstNonEmpty(
			asString(req.Token.Raw["corp_id"]),
			asString(req.Token.Raw["corpId"]),
			asString(req.Token.Raw["app_key"]),
			tenantScope,
		)
	}
	return contracts.ExternalIdentity{
		Provider:       p.Key(),
		ExternalUserID: uid,
		TenantScope:    tenantScope,
		Raw:            req.Token.Raw,
	}, nil
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
