// Package notifications provides typed access to tenant notification delivery.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/hosthttp"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type Config struct {
	BaseURL, BearerToken, STSClientID, STSClientSecret, STSTokenURL string
	AuthScheme, APIKey, PluginID                                    string
	// APIPrefix comes from Gateway.APIPrefix / PX_GATEWAY_API_PREFIX.
	APIPrefix string
	Timeout   time.Duration
}

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Client struct {
	baseURL    string
	pluginID   string
	authScheme string
	http       *http.Client
	tokens     TokenProvider
}

func (c *Client) CredentialScheme() string {
	if c == nil {
		return ""
	}
	return strings.ToLower(c.authScheme)
}

// Publisher is the Framework boundary for tenant notification delivery.
type Publisher interface {
	Create(context.Context, CreateInput) (*Notification, error)
}

// CreateInput is a tenant-scoped notification command. The target member is
// optional; omitting it delegates delivery to the host's tenant policy.
type Navigation struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type Action struct {
	ID     string            `json:"id"`
	Label  string            `json:"label"`
	Type   string            `json:"type"`
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

type CreateInput struct {
	Navigation  *Navigation    `json:"navigation,omitempty"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Type        string         `json:"type,omitempty"`
	Category    string         `json:"category,omitempty"`
	IsImportant bool           `json:"is_important,omitempty"`
	MemberUUID  string         `json:"member_uuid,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Notification struct {
	Revision       int64          `json:"revision"`
	SourceRevision int64          `json:"sourceRevision,omitempty"`
	Actions        []Action       `json:"actions,omitempty"`
	UUID           string         `json:"id"`
	Title          string         `json:"title"`
	Content        string         `json:"content"`
	Type           string         `json:"type"`
	Category       string         `json:"category"`
	IsRead         bool           `json:"isRead"`
	IsImportant    bool           `json:"isImportant"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	MemberUUID     string         `json:"userId,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type HTTPError struct {
	StatusCode int
	ReasonCode string
	ErrorCode  string
	TraceID    string
	RequestID  string
	Details    map[string]any
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx notifications request failed: reason=%s status=%d", e.ReasonCode, e.StatusCode)
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx notifications base_url is required")
	}
	c, err := newClient(cfg, httpClient)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(cfg.AuthScheme, "apikey") {
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, errors.New("NOTIFICATION_API_KEY_REQUIRED")
		}
		c.authScheme = "ApiKey"
		c.tokens = staticToken(strings.TrimSpace(cfg.APIKey))
		return c, nil
	}
	if cfg.AuthScheme != "" && !strings.EqualFold(cfg.AuthScheme, "bearer") {
		return nil, errors.New("NOTIFICATION_AUTH_SCHEME_INVALID")
	}
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		c.tokens = staticToken(token)
		return c, nil
	}
	// Installation STS uses the real bootstrap exchange through TokenProvider.
	return nil, errors.New("NOTIFICATION_STS_TOKEN_PROVIDER_REQUIRED")
}

func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if cfg.AuthScheme != "" && !strings.EqualFold(cfg.AuthScheme, "bearer") {
		return nil, errors.New("NOTIFICATION_AUTH_SCHEME_INVALID")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx notifications base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx notifications token provider is required")
	}
	c, err := newClient(cfg, httpClient)
	if err != nil {
		return nil, err
	}
	c.tokens = provider
	return c, nil
}

func newClient(cfg Config, httpClient *http.Client) (*Client, error) {
	baseURL, err := hosthttp.APIBaseURL(cfg.BaseURL, cfg.APIPrefix)
	if err != nil {
		return nil, err
	}
	c := &Client{baseURL: baseURL, pluginID: cfg.PluginID, authScheme: "Bearer"}
	if httpClient != nil {
		c.http = httpClient
		return c, nil
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	c.http = &http.Client{Timeout: timeout}
	return c, nil
}

func (c *Client) Create(ctx context.Context, input CreateInput) (*Notification, error) {
	if c == nil || c.http == nil || c.tokens == nil {
		return nil, errors.New("powerx notifications client is not configured")
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("notification title is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return nil, errors.New("notification content is required")
	}
	ref, err := CardReference(input.Metadata)
	if err != nil {
		return nil, err
	}
	if ref != nil && (!validUUID(input.MemberUUID) || (c.pluginID != "" && c.pluginID != ref.SourcePlugin)) {
		return nil, invalidCard()
	}
	var out Notification
	var response json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/notifications", input, &response); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(response, &out); err != nil {
		return nil, contractError("NOTIFICATION_RESPONSE_INVALID")
	}
	if strings.TrimSpace(out.UUID) == "" {
		return nil, contractError("NOTIFICATION_RESPONSE_INVALID")
	}
	if ref != nil {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response, &fields); err != nil {
			return nil, contractError("NOTIFICATION_CARD_RESPONSE_INVALID")
		}
		for _, key := range []string{"id", "revision", "sourceRevision", "title", "content", "type", "category", "isRead", "isImportant", "createdAt", "updatedAt", "userId", "metadata"} {
			if len(fields[key]) == 0 || bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
				return nil, contractError("NOTIFICATION_CARD_RESPONSE_INVALID")
			}
		}
		actual, err := CardReference(out.Metadata)
		if err != nil || actual == nil || *actual != *ref || !validUUID(out.UUID) || out.MemberUUID != input.MemberUUID || !validRevision(out.Revision) || !validRevision(out.SourceRevision) || out.SourceRevision < ref.SourceRevision {
			return nil, contractError("NOTIFICATION_CARD_RESPONSE_INVALID")
		}
		// The complete caller metadata must survive creation, not only business_card.
		for key, value := range input.Metadata {
			expectedJSON, _ := json.Marshal(value)
			var expected any
			_ = json.Unmarshal(expectedJSON, &expected)
			received, present := out.Metadata[key]
			if !present || !reflect.DeepEqual(expected, received) {
				return nil, contractError("NOTIFICATION_CARD_RESPONSE_INVALID")
			}
		}
	}
	if input.Navigation != nil {
		matched := false
		for _, action := range out.Actions {
			if action.Type == "primary" && action.Action == "navigate" && action.Label == input.Navigation.Label && action.Params["path"] == input.Navigation.Path {
				matched = true
				break
			}
		}
		if !matched {
			return nil, &HTTPError{StatusCode: 502, ReasonCode: "NOTIFICATION_NAVIGATION_RESPONSE_INVALID"}
		}
	}
	return &out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &HTTPError{StatusCode: 503, ReasonCode: "NOTIFICATION_UPSTREAM_DEPENDENCY"}
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("powerx notifications token provider returned an empty token")
	}
	req.Header.Set("Authorization", c.authScheme+" "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &HTTPError{StatusCode: 503, ReasonCode: "NOTIFICATION_UPSTREAM_DEPENDENCY"}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &HTTPError{StatusCode: 502, ReasonCode: "NOTIFICATION_UPSTREAM_DEPENDENCY"}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DecodeHTTPError(resp.StatusCode, resp.Header, raw)
	}
	if err := hostcontract.DecodeData(raw, output); err != nil {
		failure := DecodeHTTPError(502, resp.Header, raw)
		failure.ReasonCode = "NOTIFICATION_RESPONSE_INVALID"
		return failure
	}
	return nil
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }
