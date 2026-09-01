// Package integration is the typed tenant Integration Gateway client.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
)

type Config struct {
	BaseURL, BearerToken, STSClientID, STSClientSecret, STSTokenURL string
	Timeout                                                         time.Duration
}
type TokenProvider interface {
	Token(context.Context) (string, error)
}
type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenProvider
}

type RouteSummary struct {
	RouteUUID      string   `json:"route_id"`
	RouteSlug      string   `json:"route_slug"`
	CapabilityID   string   `json:"capability_id"`
	Channels       []string `json:"channels"`
	LifecycleState string   `json:"lifecycle_state"`
	Status         string   `json:"status"`
	UpdatedAt      string   `json:"updated_at"`
}
type RateLimitPolicy struct {
	Limit         uint64 `json:"limit"`
	Burst         uint64 `json:"burst"`
	WindowSeconds int    `json:"window_seconds"`
	Scope         string `json:"scope"`
}
type RouteDetail struct {
	RouteUUID      string          `json:"route_id"`
	RouteSlug      string          `json:"route_slug"`
	CapabilityID   string          `json:"capability_id"`
	ToolGrantIDs   []string        `json:"tool_grant_ids"`
	Channels       []string        `json:"channels"`
	RateLimit      RateLimitPolicy `json:"rate_limit"`
	LifecycleState string          `json:"lifecycle_state"`
	Status         string          `json:"status"`
	Description    string          `json:"description,omitempty"`
	CurrentVersion uint32          `json:"current_version"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}
type ListRoutesInput struct{ CapabilityID, Channel string }
type InvokeRouteInput struct {
	Payload        map[string]any `json:"payload"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Context        map[string]any `json:"context,omitempty"`
}
type InvokeRouteOutput struct {
	Result             map[string]any `json:"result,omitempty"`
	RoutedCapabilityID string         `json:"routed_capability_id"`
	RoutedAdapter      string         `json:"routed_adapter,omitempty"`
	TraceID            string         `json:"trace_id"`
	DispatchedAt       string         `json:"dispatched_at"`
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx integration base_url is required")
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
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		c.tokens = staticToken(token)
		return c, nil
	}
	client, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, fmt.Errorf("powerx integration delegated sts configuration: %w", err)
	}
	c.tokens = stsToken{client}
	return c, nil
}

func (c *Client) ListRoutes(ctx context.Context, input ListRoutesInput) ([]RouteSummary, error) {
	query := url.Values{}
	if v := strings.TrimSpace(input.CapabilityID); v != "" {
		query.Set("capability_id", v)
	}
	if v := strings.TrimSpace(input.Channel); v != "" {
		query.Set("channel", v)
	}
	path := "/api/v1/tenant/integration/routes"
	if q := query.Encode(); q != "" {
		path += "?" + q
	}
	var raw struct {
		Items []RouteSummary `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Items, nil
}
func (c *Client) GetRoute(ctx context.Context, routeSlug string) (*RouteDetail, error) {
	var out RouteDetail
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/tenant/integration/routes/"+url.PathEscape(strings.TrimSpace(routeSlug)), nil, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.RouteUUID) == "" {
		return nil, errors.New("powerx integration response missing route uuid")
	}
	return &out, nil
}
func (c *Client) InvokeRoute(ctx context.Context, routeSlug string, input InvokeRouteInput) (*InvokeRouteOutput, error) {
	if len(input.Payload) == 0 {
		return nil, errors.New("powerx integration payload is required")
	}
	var out InvokeRouteOutput
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/tenant/integration/routes/"+url.PathEscape(strings.TrimSpace(routeSlug))+"/invoke", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, input, output any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("powerx integration client is not configured")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(payload)}
	}
	if output == nil {
		return nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return errors.New("powerx integration response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx integration request failed: status=%d", e.StatusCode)
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) {
	if strings.TrimSpace(string(t)) == "" {
		return "", errors.New("powerx integration bearer token is required")
	}
	return string(t), nil
}

type stsToken struct{ client *sts.Client }

func (t stsToken) Token(ctx context.Context) (string, error) {
	if t.client == nil {
		return "", errors.New("powerx integration sts client is required")
	}
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
