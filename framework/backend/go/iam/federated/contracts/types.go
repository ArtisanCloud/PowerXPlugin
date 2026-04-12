package contracts

import "time"

// AuthorizeRequest 是发起扫码授权时的标准输入。
type AuthorizeRequest struct {
	TenantUUID  string
	RedirectURI string
	State       string
	Nonce       string
	Scope       string
}

// AuthorizeResponse 是发起扫码授权时的标准输出。
type AuthorizeResponse struct {
	AuthorizeURL string
}

// ExchangeCodeRequest 描述 code 兑换请求。
type ExchangeCodeRequest struct {
	Code string
}

// ProviderToken 描述 provider 兑换后的令牌。
type ProviderToken struct {
	AccessToken string
	IDToken     string
	Raw         map[string]any
}

// ResolveIdentityRequest 描述如何从 provider token 解析身份。
type ResolveIdentityRequest struct {
	Token ProviderToken
}

// ExternalIdentity 是统一外部身份定义。
type ExternalIdentity struct {
	Provider       string
	ExternalUserID string
	UnionID        string
	OpenID         string
	Email          string
	Phone          string
	TenantScope    string
	Raw            map[string]any
}

// LoginChallenge 描述一次扫码登录挑战。
type LoginChallenge struct {
	State      string
	Nonce      string
	TenantUUID string
	Provider   string
	TraceID    string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

// ChallengeIssueRequest 描述 challenge 签发入参。
type ChallengeIssueRequest struct {
	TenantUUID string
	Provider   string
	TraceID    string
	TTL        time.Duration
	Now        time.Time
}

// ChallengeConsumeRequest 描述 challenge 校验与消费入参。
type ChallengeConsumeRequest struct {
	State      string
	Nonce      string
	TenantUUID string
	Provider   string
	Now        time.Time
}

// RiskEvaluateRequest 描述回调风险判定输入。
type RiskEvaluateRequest struct {
	Challenge      LoginChallenge
	State          string
	Nonce          string
	Code           string
	TenantUUID     string
	SignatureValid bool
	Now            time.Time
}

// RiskDecision 描述风险判定输出。
type RiskDecision struct {
	Allowed  bool
	Code     ErrorCode
	Reason   string
	Evidence map[string]string
}
