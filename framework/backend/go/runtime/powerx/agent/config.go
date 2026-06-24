package agent

import (
	"os"
	"strings"
	"time"
)

const (
	ModeStandalone = "standalone"
	ModeDelegated  = "delegated"
	AuthBearer     = "bearer"
	DefaultTimeout  = 5 * time.Minute
)

type PowerXAgentClientConfig struct {
	PluginID        string
	Mode            string
	BaseURL         string
	InvokePath      string
	SSEPath         string
	WSPath          string
	AuthScheme      string
	BearerToken     string
	STSClientID     string
	STSClientSecret string
	STSTokenURL     string
	Timeout         time.Duration
	ReconnectPolicy ReconnectPolicy
}

type ReconnectPolicy struct {
	Enabled     bool
	MaxAttempts int
	Backoff     time.Duration
}

func (c PowerXAgentClientConfig) WithDefaults() PowerXAgentClientConfig {
	if c.Mode == "" {
		c.Mode = ModeStandalone
	}
	if c.InvokePath == "" {
		c.InvokePath = "/api/v1/agents/invoke"
	}
	if c.SSEPath == "" {
		c.SSEPath = "/api/v1/agents/stream/sse"
	}
	if c.WSPath == "" {
		c.WSPath = "/api/v1/agents/stream/ws"
	}
	if c.AuthScheme == "" {
		c.AuthScheme = AuthBearer
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.ReconnectPolicy.MaxAttempts == 0 {
		c.ReconnectPolicy.MaxAttempts = 3
	}
	if c.ReconnectPolicy.Backoff <= 0 {
		c.ReconnectPolicy.Backoff = time.Second
	}
	return c
}

func ValidateConfig(c PowerXAgentClientConfig) error {
	c = c.WithDefaults()
	if strings.TrimSpace(c.BaseURL) == "" {
		return newError(ErrCodeConfigInvalid, "base_url is required")
	}
	if c.Mode == ModeDelegated {
		if strings.ToLower(c.AuthScheme) != AuthBearer {
			return newError(ErrCodeAuthInvalid, "delegated mode requires bearer auth_scheme")
		}
		if strings.TrimSpace(c.BearerToken) != "" {
			if strings.EqualFold(strings.TrimSpace(os.Getenv("PX_TOOL_TOKEN")), c.BearerToken) {
				return newError(ErrCodeAuthInvalid, "delegated mode must not use PX_TOOL_TOKEN")
			}
			if strings.EqualFold(strings.TrimSpace(os.Getenv("PX_GATEWAY_API_KEY")), c.BearerToken) {
				return newError(ErrCodeAuthInvalid, "delegated mode must not use PX_GATEWAY_API_KEY")
			}
			return newError(ErrCodeAuthInvalid, "delegated mode requires STS config; static bearer token is forbidden")
		}
		if c.STSClientID == "" || c.STSClientSecret == "" || c.STSTokenURL == "" {
			return newError(ErrCodeAuthInvalid, "delegated mode requires STS config")
		}
		return nil
	}
	if c.BearerToken == "" && (c.STSClientID == "" || c.STSClientSecret == "" || c.STSTokenURL == "") {
		return newError(ErrCodeAuthInvalid, "bearer token or STS config is required")
	}
	return nil
}
