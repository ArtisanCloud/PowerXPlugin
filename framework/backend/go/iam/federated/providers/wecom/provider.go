package wecom

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

// Provider 是企业微信默认 provider 实现（可被插件覆写配置）。
type Provider struct {
	corpID string
}

func New(corpID string) *Provider {
	return &Provider{corpID: strings.TrimSpace(corpID)}
}

func (p *Provider) Key() string { return "wecom" }

func (p *Provider) BuildAuthorizeURL(_ context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	q := url.Values{}
	q.Set("appid", p.corpID)
	q.Set("redirect_uri", req.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", firstNonEmpty(req.Scope, "snsapi_base"))
	q.Set("state", req.State)
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	return contracts.AuthorizeResponse{AuthorizeURL: "https://open.weixin.qq.com/connect/oauth2/authorize?" + q.Encode()}, nil
}

func (p *Provider) ExchangeCode(_ context.Context, req contracts.ExchangeCodeRequest) (contracts.ProviderToken, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "code is required")
	}
	return contracts.ProviderToken{AccessToken: "wecom-token-" + code, Raw: map[string]any{"code": code}}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("wecom:%s", token)
	return contracts.ExternalIdentity{Provider: p.Key(), ExternalUserID: uid, Raw: req.Token.Raw}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
