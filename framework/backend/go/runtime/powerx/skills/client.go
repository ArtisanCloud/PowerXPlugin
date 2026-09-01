package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
)

type Config struct {
	BaseURL, BearerToken, STSClientID, STSClientSecret, STSTokenURL string
	Timeout                                                         time.Duration
}
type InvokeInput struct {
	SkillID string         `json:"skill_id"`
	Version string         `json:"version,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}
type InvokeOutput struct {
	TraceID      string         `json:"trace_id"`
	Status       string         `json:"status"`
	ProtocolUsed string         `json:"protocol_used"`
	FallbackUsed bool           `json:"fallback_used"`
	Result       map[string]any `json:"result,omitempty"`
}
type Client struct {
	baseURL string
	http    *http.Client
	token   func(context.Context) (string, error)
}

// Invoker is the Framework boundary for direct tenant skill invocation.
type Invoker interface {
	Invoke(context.Context, InvokeInput) (*InvokeOutput, error)
}

// TokenProvider supplies the tenant-scoped service token used for delegated
// calls. It is intentionally shared with the other PowerX runtime clients.
type TokenProvider interface {
	Token(context.Context) (string, error)
}

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

// HTTPError preserves the failed HTTP status without treating an upstream
// authorization or validation failure as a transport success.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx skills request failed: status=%d", e.StatusCode)
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx skills base_url is required")
	}
	c := &Client{baseURL: strings.TrimRight(cfg.BaseURL, "/")}
	if httpClient != nil {
		c.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		c.http = &http.Client{Timeout: timeout}
	}
	if t := strings.TrimSpace(cfg.BearerToken); t != "" {
		c.token = func(context.Context) (string, error) { return t, nil }
		return c, nil
	}
	stsClient, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, err
	}
	c.token = func(ctx context.Context) (string, error) {
		token, err := stsClient.Token(ctx)
		return token.AccessToken, err
	}
	return c, nil
}

// NewClientWithTokenProvider builds a delegated client from the runtime's
// shared STS token provider. It avoids a second, divergent STS configuration.
func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx skills base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx skills token provider is required")
	}
	c := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")}
	if httpClient != nil {
		c.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		c.http = &http.Client{Timeout: timeout}
	}
	c.token = provider.Token
	return c, nil
}
func (c *Client) Invoke(ctx context.Context, in InvokeInput) (*InvokeOutput, error) {
	if c == nil || c.http == nil || c.token == nil {
		return nil, errors.New("powerx skills client is not configured")
	}
	if strings.TrimSpace(in.SkillID) == "" {
		return nil, errors.New("skill_id is required")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/tenant/skills/invoke", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, errors.New("powerx skills response missing data")
	}
	var out InvokeOutput
	if err := json.Unmarshal(envelope.Data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
