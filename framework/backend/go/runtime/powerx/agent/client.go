package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
)

type Client struct {
	cfg    PowerXAgentClientConfig
	http   *http.Client
	tokens TokenProvider
}

func NewClient(cfg PowerXAgentClientConfig, opts ...func(*Client)) (*Client, error) {
	cfg = cfg.WithDefaults()
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	c := &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
	if cfg.BearerToken != "" {
		c.tokens = StaticBearerTokenProvider{TokenValue: cfg.BearerToken}
	} else {
		stsClient, err := sts.NewClient(sts.Config{
			TokenEndpoint: cfg.STSTokenURL,
			ClientID:      cfg.STSClientID,
			ClientSecret:  cfg.STSClientSecret,
		})
		if err != nil {
			return nil, err
		}
		c.tokens = STSTokenProvider{Client: stsClient}
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func WithHTTPClient(httpClient *http.Client) func(*Client) {
	return func(c *Client) {
		if httpClient != nil {
			c.http = httpClient
		}
	}
}

func WithTokenProvider(provider TokenProvider) func(*Client) {
	return func(c *Client) {
		if provider != nil {
			c.tokens = provider
		}
	}
}

func (c *Client) Invoke(ctx context.Context, req AgentInvokeRequest) (AgentInvokeResponse, error) {
	var out AgentInvokeResponse
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(c.cfg.InvokePath), bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, httpReq); err != nil {
		return out, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return out, &Error{Code: ErrCodeTransport, Message: resp.Status}
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) authorize(ctx context.Context, req *http.Request) error {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (c *Client) url(path string) string {
	base := strings.TrimRight(c.cfg.BaseURL, "/")
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "ws://") || strings.HasPrefix(path, "wss://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func withQuery(raw string, values url.Values) string {
	if len(values) == 0 {
		return raw
	}
	sep := "?"
	if strings.Contains(raw, "?") {
		sep = "&"
	}
	return raw + sep + values.Encode()
}
