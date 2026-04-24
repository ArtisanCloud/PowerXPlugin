package wecom

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

// Provider 是企业微信默认 provider 实现（可被插件覆写配置）。
type Provider struct {
	corpID   string
	agentID  int
	secret   string
	token    string
	aesKey   string
	callback string
	resolver ConfigResolver
}

type Config struct {
	CorpID      string
	AgentID     int
	Secret      string
	Token       string
	AESKey      string
	CallbackURL string
	HttpDebug   bool
}

type ConfigResolver func(ctx context.Context, tenantUUID string) (Config, error)

func New(corpID string) *Provider {
	return &Provider{corpID: strings.TrimSpace(corpID)}
}

func NewWithConfig(cfg Config) *Provider {
	return &Provider{
		corpID:   strings.TrimSpace(cfg.CorpID),
		agentID:  cfg.AgentID,
		secret:   strings.TrimSpace(cfg.Secret),
		token:    strings.TrimSpace(cfg.Token),
		aesKey:   strings.TrimSpace(cfg.AESKey),
		callback: strings.TrimSpace(cfg.CallbackURL),
	}
}

func NewWithResolver(resolver ConfigResolver) *Provider {
	return &Provider{resolver: resolver}
}

func (p *Provider) Key() string { return "wecom" }

func (p *Provider) BuildAuthorizeURL(ctx context.Context, req contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	if strings.TrimSpace(req.RedirectURI) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "redirect uri is required")
	}
	cfg, err := p.resolveConfig(ctx, req.TenantUUID)
	if err != nil {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, err.Error())
	}
	if strings.TrimSpace(cfg.CorpID) == "" {
		return contracts.AuthorizeResponse{}, contracts.NewError(contracts.ErrorCodeProviderNotFound, "missing wecom corp_id")
	}
	q := url.Values{}
	q.Set("appid", cfg.CorpID)
	q.Set("redirect_uri", req.RedirectURI)
	if cfg.AgentID > 0 {
		q.Set("agentid", strconv.Itoa(cfg.AgentID))
	}
	q.Set("state", req.State)
	return contracts.AuthorizeResponse{AuthorizeURL: "https://open.work.weixin.qq.com/wwopen/sso/qrConnect?" + q.Encode()}, nil
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
	if strings.TrimSpace(cfg.Secret) == "" || strings.TrimSpace(cfg.CorpID) == "" || cfg.AgentID <= 0 {
		// 兼容旧测试与未配置场景：未提供 WeCom 凭据时保持 mock 行为。
		return contracts.ProviderToken{AccessToken: "wecom-token-" + code, Raw: map[string]any{"code": code}}, nil
	}
	app, err := p.newWorkOAuthApp(cfg)
	if err != nil {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeUnauthorized, err.Error())
	}
	userInfo, err := app.Auth.GetUserInfo(ctx, code)
	if err != nil {
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeUnauthorized, err.Error())
	}
	if userInfo == nil || userInfo.ErrCode != 0 {
		msg := "wecom auth failed"
		if userInfo != nil {
			msg = fmt.Sprintf("wecom auth failed: %d %s", userInfo.ErrCode, userInfo.ErrMsg)
		}
		return contracts.ProviderToken{}, contracts.NewError(contracts.ErrorCodeUnauthorized, msg)
	}
	raw := map[string]any{
		"code":        code,
		"corp_id":     strings.TrimSpace(cfg.CorpID),
		"user_id":     strings.TrimSpace(userInfo.UserID),
		"user_ticket": strings.TrimSpace(userInfo.UserTicket),
		"open_id":     strings.TrimSpace(userInfo.OpenID),
		"external_id": strings.TrimSpace(userInfo.ExternalUserID),
	}
	token := strings.TrimSpace(userInfo.UserID)
	if token == "" {
		token = code
	}
	return contracts.ProviderToken{AccessToken: "wecom-token-" + token, Raw: raw}, nil
}

func (p *Provider) ResolveIdentity(_ context.Context, req contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	tenantScope := strings.TrimSpace(p.corpID)
	if req.Token.Raw != nil {
		tenantScope = firstNonEmpty(asString(req.Token.Raw["corp_id"]), tenantScope)
	}
	if req.Token.Raw != nil {
		if ext := firstNonEmpty(
			asString(req.Token.Raw["external_id"]),
			asString(req.Token.Raw["user_id"]),
			asString(req.Token.Raw["open_id"]),
		); ext != "" {
			return contracts.ExternalIdentity{
				Provider:       p.Key(),
				ExternalUserID: fmt.Sprintf("wecom:%s", ext),
				TenantScope:    tenantScope,
				Raw:            req.Token.Raw,
			}, nil
		}
	}
	token := strings.TrimSpace(req.Token.AccessToken)
	if token == "" {
		return contracts.ExternalIdentity{}, contracts.NewError(contracts.ErrorCodeUnauthorized, "missing access token")
	}
	uid := fmt.Sprintf("wecom:%s", token)
	return contracts.ExternalIdentity{
		Provider:       p.Key(),
		ExternalUserID: uid,
		TenantScope:    tenantScope,
		Raw:            req.Token.Raw,
	}, nil
}

func (p *Provider) newWorkOAuthApp(cfg Config) (*work.Work, error) {
	if strings.TrimSpace(cfg.CorpID) == "" || strings.TrimSpace(cfg.Secret) == "" || cfg.AgentID <= 0 {
		return nil, fmt.Errorf("missing wecom oauth credentials")
	}
	return work.NewWork(&work.UserConfig{
		CorpID:      strings.TrimSpace(cfg.CorpID),
		AgentID:     cfg.AgentID,
		Secret:      strings.TrimSpace(cfg.Secret),
		Token:       strings.TrimSpace(cfg.Token),
		AESKey:      strings.TrimSpace(cfg.AESKey),
		CallbackURL: strings.TrimSpace(cfg.CallbackURL),
		OAuth: work.OAuth{
			Callback: strings.TrimSpace(cfg.CallbackURL),
			Scopes:   []string{"snsapi_privateinfo"},
		},
		HttpDebug: cfg.HttpDebug,
	})
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
		CorpID:      p.corpID,
		AgentID:     p.agentID,
		Secret:      p.secret,
		Token:       p.token,
		AESKey:      p.aesKey,
		CallbackURL: p.callback,
		HttpDebug:   false,
	}), nil
}

func normalizeConfig(cfg Config) Config {
	cfg.CorpID = strings.TrimSpace(cfg.CorpID)
	cfg.Secret = strings.TrimSpace(cfg.Secret)
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.AESKey = strings.TrimSpace(cfg.AESKey)
	cfg.CallbackURL = strings.TrimSpace(cfg.CallbackURL)
	return cfg
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatInt(int64(t), 10)
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
