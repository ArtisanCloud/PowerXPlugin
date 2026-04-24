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
	appID     string
	appSecret string
	tenantKey string
	callback  string
	resolver  ConfigResolver
}

type Config struct {
	AppID       string
	AppSecret   string
	TenantKey   string
	CallbackURL string
}

type ConfigResolver func(ctx context.Context, tenantUUID string) (Config, error)

func New(appID string) *Provider {
	return &Provider{appID: strings.TrimSpace(appID)}
}

func NewWithConfig(cfg Config) *Provider {
	return &Provider{
		appID:     strings.TrimSpace(cfg.AppID),
		appSecret: strings.TrimSpace(cfg.AppSecret),
		tenantKey: strings.TrimSpace(cfg.TenantKey),
		callback:  strings.TrimSpace(cfg.CallbackURL),
	}
}

func NewWithResolver(resolver ConfigResolver) *Provider {
	return &Provider{resolver: resolver}
}

func (p *Provider) Key() string { return "lark" }

func (p *Provider) BuildAuthorizeURL(ctx context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	cfg, err := p.resolveConfig(ctx, req.TenantUUID)
	if err != nil {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, err.Error())
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, "missing lark app_id")
	}
	q := url.Values{}
	q.Set("app_id", cfg.AppID)
	q.Set("redirect_uri", req.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", firstNonEmpty(req.Scope, "contact:user.base:readonly"))
	q.Set("state", req.State)
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	return contracts.AuthorizeResponse{AuthorizeURL: "https://accounts.feishu.cn/open-apis/authen/v1/authorize?" + q.Encode()}, nil
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
		AccessToken: "lark-token-" + code,
		Raw: map[string]any{
			"code":       code,
			"app_id":     strings.TrimSpace(cfg.AppID),
			"app_secret": strings.TrimSpace(cfg.AppSecret),
			"tenant_key": strings.TrimSpace(cfg.TenantKey),
		},
	}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("lark:%s", token)
	tenantScope := strings.TrimSpace(p.tenantKey)
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
		AppID:       p.appID,
		AppSecret:   p.appSecret,
		TenantKey:   p.tenantKey,
		CallbackURL: p.callback,
	}), nil
}

func normalizeConfig(cfg Config) Config {
	cfg.AppID = strings.TrimSpace(cfg.AppID)
	cfg.AppSecret = strings.TrimSpace(cfg.AppSecret)
	cfg.TenantKey = strings.TrimSpace(cfg.TenantKey)
	cfg.CallbackURL = strings.TrimSpace(cfg.CallbackURL)
	return cfg
}
