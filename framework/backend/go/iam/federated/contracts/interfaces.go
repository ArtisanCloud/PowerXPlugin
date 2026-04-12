package contracts

import "context"

// Provider 定义渠道登录 provider 的最小协议。
type Provider interface {
	Key() string
	BuildAuthorizeURL(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error)
	ExchangeCode(ctx context.Context, req ExchangeCodeRequest) (ProviderToken, error)
	ResolveIdentity(ctx context.Context, req ResolveIdentityRequest) (ExternalIdentity, error)
}

// ProviderFactory 提供 provider 的注册与检索。
type ProviderFactory interface {
	Register(provider Provider) error
	Get(key string) (Provider, bool)
	MustGet(key string) (Provider, error)
	List() []string
}

// ChallengeManager 管理扫码 challenge 的生命周期。
type ChallengeManager interface {
	Issue(ctx context.Context, req ChallengeIssueRequest) (LoginChallenge, error)
	ValidateAndConsume(ctx context.Context, req ChallengeConsumeRequest) (LoginChallenge, error)
}

// RiskEvaluator 对回调请求进行风险判定。
type RiskEvaluator interface {
	EvaluateCallback(ctx context.Context, req RiskEvaluateRequest) RiskDecision
}
