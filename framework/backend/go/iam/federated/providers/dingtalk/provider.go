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
	appKey    string
	appSecret string
	corpID    string
	callback  string
	resolver  ConfigResolver
}

type Config struct {
	AppKey      string
	AppSecret   string
	CorpID      string
	CallbackURL string
}

type ConfigResolver func(ctx context.Context, tenantUUID string) (Config, error)

func New(appKey string) *Provider {
	return &Provider{appKey: strings.TrimSpace(appKey)}
}

func NewWithConfig(cfg Config) *Provider {
	return &Provider{
		appKey:    strings.TrimSpace(cfg.AppKey),
		appSecret: strings.TrimSpace(cfg.AppSecret),
		corpID:    strings.TrimSpace(cfg.CorpID),
		callback:  strings.TrimSpace(cfg.CallbackURL),
	}
}

func NewWithResolver(resolver ConfigResolver) *Provider {
	return &Provider{resolver: resolver}
}

func (p *Provider) Key() string { return "dingtalk" }

func (p *Provider) BuildAuthorizeURL(ctx context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	cfg, err := p.resolveConfig(ctx, req.TenantUUID)
	if err != nil {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, err.Error())
	}
	if strings.TrimSpace(cfg.AppKey) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, "missing dingtalk app_key")
	}
	q := url.Values{}
	q.Set("client_id", cfg.AppKey)
	q.Set("redirect_uri", req.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", firstNonEmpty(req.Scope, "openid"))
	q.Set("state", req.State)
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	return contracts.AuthorizeResponse{AuthorizeURL: "https://login.dingtalk.com/oauth2/auth?" + q.Encode()}, nil
}

func (p *Provider) ExchangeCode(ctx context.Context, req contracts.ExchangeCodeRequest) (contracts.ProviderToken, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "code is required")
	}
	cfg, err := p.resolveConfig(ctx, req.TenantUUID)
	if err != nil {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, err.Error())
	}
	return contracts.ProviderToken{
		AccessToken: "dingtalk-token-" + code,
		Raw: map[string]any{
			"code":       code,
			"app_key":    strings.TrimSpace(cfg.AppKey),
			"app_secret": strings.TrimSpace(cfg.AppSecret),
			"corp_id":    strings.TrimSpace(cfg.CorpID),
		},
	}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("dingtalk:%s", token)
	tenantScope := strings.TrimSpace(p.corpID)
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

func (p *Provider) resolveConfig(ctx context.Context, tenantUUID string) (Config, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if p.resolver != nil && tenantUUID != "" {
		cfg, err := p.resolver(ctx, tenantUUID)
		if err != nil {
			return Config{}, err
		}
		return normalizeConfig(cfg), nil
	}
	return normalizeConfig(Config{
		AppKey:      p.appKey,
		AppSecret:   p.appSecret,
		CorpID:      p.corpID,
		CallbackURL: p.callback,
	}), nil
}

func normalizeConfig(cfg Config) Config {
	cfg.AppKey = strings.TrimSpace(cfg.AppKey)
	cfg.AppSecret = strings.TrimSpace(cfg.AppSecret)
	cfg.CorpID = strings.TrimSpace(cfg.CorpID)
	cfg.CallbackURL = strings.TrimSpace(cfg.CallbackURL)
	return cfg
}
