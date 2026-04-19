package lark

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

// Provider 是飞书默认 provider 实现（可被插件覆写配置）。
type Provider struct {
	appID string
}

func New(appID string) *Provider {
	return &Provider{appID: strings.TrimSpace(appID)}
}

func (p *Provider) Key() string { return "lark" }

func (p *Provider) BuildAuthorizeURL(_ context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	q := url.Values{}
	q.Set("app_id", p.appID)
	q.Set("redirect_uri", req.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", firstNonEmpty(req.Scope, "contact:user.base:readonly"))
	q.Set("state", req.State)
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	return contracts.AuthorizeResponse{AuthorizeURL: "https://accounts.feishu.cn/open-apis/authen/v1/authorize?" + q.Encode()}, nil
}

func (p *Provider) ExchangeCode(_ context.Context, req contracts.ExchangeCodeRequest) (contracts.ProviderToken, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "code is required")
	}
	return contracts.ProviderToken{
		AccessToken: "lark-token-" + code,
		Raw: map[string]any{
			"code":   code,
			"app_id": strings.TrimSpace(p.appID),
		},
	}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("lark:%s", token)
	tenantScope := strings.TrimSpace(p.appID)
	if req.Token.Raw != nil {
		tenantScope = firstNonEmpty(
			asString(req.Token.Raw["tenant_key"]),
			asString(req.Token.Raw["enterprise_id"]),
			asString(req.Token.Raw["app_id"]),
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
