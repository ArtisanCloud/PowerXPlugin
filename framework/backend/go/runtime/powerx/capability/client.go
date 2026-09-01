// Package capability provides typed access to the tenant Capability Registry.
package capability

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
type tokenProvider interface {
	Token(context.Context) (string, error)
}

// TokenProvider supplies the tenant-scoped service token used for delegated
// calls. It is exported so Skeleton can share its already-configured STS flow.
type TokenProvider = tokenProvider

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Client struct {
	baseURL string
	http    *http.Client
	token   tokenProvider
}

// Registry is the Framework boundary for tenant Capability Registry access.
// Plugin business services should depend on this interface, not on Client.
type Registry interface {
	List(context.Context, ListInput) ([]Capability, error)
	Resolve(context.Context, ResolveInput) (*ResolveResult, error)
	Invoke(context.Context, InvokeInput) (*InvokeResult, error)
	GetInvocation(context.Context, string) (*Invocation, error)
}

// HTTPError preserves the failed HTTP status so callers can apply the Core
// capability contract's authorization and retry semantics.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx capability request failed: status=%d", e.StatusCode)
}

type Protocol struct {
	Channel       string  `json:"channel"`
	Endpoint      string  `json:"endpoint,omitempty"`
	Method        string  `json:"method,omitempty"`
	RPC           string  `json:"rpc,omitempty"`
	SchemaRef     string  `json:"schema_ref,omitempty"`
	ToolRef       string  `json:"tool_ref,omitempty"`
	AuthType      string  `json:"auth_type,omitempty"`
	HealthState   string  `json:"health_state,omitempty"`
	LastCheckedAt *string `json:"last_checked_at,omitempty"`
}
type Policy struct {
	Prefer             string   `json:"prefer,omitempty"`
	Fallback           []string `json:"fallback,omitempty"`
	RollbackCapability string   `json:"rollback_capability_id,omitempty"`
}
type Capability struct {
	CapabilityID     string     `json:"capability_id"`
	PluginID         string     `json:"plugin_id"`
	PluginVersion    string     `json:"plugin_version"`
	Title            string     `json:"title"`
	Description      string     `json:"description,omitempty"`
	Source           string     `json:"source"`
	Categories       []string   `json:"categories,omitempty"`
	Intents          []string   `json:"intents,omitempty"`
	ToolScopes       []string   `json:"tool_scope,omitempty"`
	Policy           *Policy    `json:"policy,omitempty"`
	Protocols        []Protocol `json:"protocols"`
	CapabilitiesHash string     `json:"capabilities_hash"`
	ProtocolHash     string     `json:"protocol_hash"`
	Status           string     `json:"status"`
	PublishedAt      *string    `json:"published_at,omitempty"`
}
type ListInput struct {
	Page, PageSize                                int
	PluginID, Intent, ToolScope, Protocol, Source string
}
type ResolveInput struct{ Method, Endpoint, Source string }
type ResolveResult struct {
	CapabilityID    string `json:"capability_id"`
	PluginID        string `json:"plugin_id"`
	Source          string `json:"source"`
	Protocol        string `json:"protocol"`
	Method          string `json:"method"`
	PatternEndpoint string `json:"pattern_endpoint"`
}
type InvokeInput struct {
	CapabilityID      string         `json:"capability_id,omitempty"`
	Intent            string         `json:"intent,omitempty"`
	ToolScope         string         `json:"tool_scope,omitempty"`
	PreferredProtocol string         `json:"preferred_protocol,omitempty"`
	IdempotencyKey    string         `json:"idempotency_key,omitempty"`
	TraceID           string         `json:"trace_id,omitempty"`
	ToolGrantIDs      []string       `json:"tool_grant_ids,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
	Context           map[string]any `json:"context,omitempty"`
}
type InvokeResult struct {
	TraceID      string         `json:"trace_id"`
	Status       string         `json:"status"`
	ProtocolUsed string         `json:"protocol_used"`
	FallbackUsed bool           `json:"fallback_used"`
	Payload      map[string]any `json:"payload,omitempty"`
	Result       map[string]any `json:"result,omitempty"`
}
type InvocationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Invocation struct {
	TraceID        string           `json:"trace_id"`
	TenantUUID     string           `json:"tenant_uuid"`
	CapabilityID   string           `json:"capability_id"`
	ProtocolUsed   string           `json:"protocol_used"`
	FallbackUsed   bool             `json:"fallback_used"`
	Status         string           `json:"status"`
	Error          *InvocationError `json:"error,omitempty"`
	LatencyMS      int              `json:"latency_ms"`
	EventPublished bool             `json:"event_published"`
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx capability base_url is required")
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
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		c.token = staticToken(token)
		return c, nil
	}
	client, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, err
	}
	c.token = stsToken{client}
	return c, nil
}

// NewClientWithTokenProvider builds a delegated client from the runtime's
// shared STS token provider rather than configuring an independent STS flow.
func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx capability base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx capability token provider is required")
	}
	c := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), token: provider}
	if httpClient != nil {
		c.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		c.http = &http.Client{Timeout: timeout}
	}
	return c, nil
}
func (c *Client) List(ctx context.Context, in ListInput) ([]Capability, error) {
	q := url.Values{}
	if in.Page > 0 {
		q.Set("page", fmt.Sprint(in.Page))
	}
	if in.PageSize > 0 {
		q.Set("page_size", fmt.Sprint(in.PageSize))
	}
	for k, v := range map[string]string{"plugin_id": in.PluginID, "intent": in.Intent, "channel": in.ToolScope, "protocol": in.Protocol, "source": in.Source} {
		if strings.TrimSpace(v) != "" {
			q.Set(k, v)
		}
	}
	var raw struct {
		Items []Capability `json:"items"`
	}
	path := "/api/v1/tenant/capabilities"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Items, nil
}
func (c *Client) Resolve(ctx context.Context, in ResolveInput) (*ResolveResult, error) {
	if strings.TrimSpace(in.Method) == "" || strings.TrimSpace(in.Endpoint) == "" {
		return nil, errors.New("capability method and endpoint are required")
	}
	q := url.Values{"method": []string{strings.ToUpper(strings.TrimSpace(in.Method))}, "endpoint": []string{strings.TrimSpace(in.Endpoint)}}
	if strings.TrimSpace(in.Source) != "" {
		q.Set("source", strings.TrimSpace(in.Source))
	}
	var raw struct {
		Primary ResolveResult `json:"primary_match"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/tenant/capabilities/resolve?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	return &raw.Primary, nil
}
func (c *Client) Invoke(ctx context.Context, in InvokeInput) (*InvokeResult, error) {
	if strings.TrimSpace(in.CapabilityID) == "" && strings.TrimSpace(in.Intent) == "" {
		return nil, errors.New("capability_id or intent is required")
	}
	var out InvokeResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/tenant/invocations", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) GetInvocation(ctx context.Context, traceID string) (*Invocation, error) {
	if strings.TrimSpace(traceID) == "" {
		return nil, errors.New("invocation trace_id is required")
	}
	var out Invocation
	if err := c.do(ctx, http.MethodGet, "/api/v1/tenant/invocations/"+url.PathEscape(strings.TrimSpace(traceID)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	if c == nil || c.http == nil || c.token == nil {
		return errors.New("powerx capability client is not configured")
	}
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	token, err := c.token.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(raw)}
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return errors.New("powerx capability response missing data")
	}
	return json.Unmarshal(envelope.Data, out)
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

type stsToken struct{ client *sts.Client }

func (t stsToken) Token(ctx context.Context) (string, error) {
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
