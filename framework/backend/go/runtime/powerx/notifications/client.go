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

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenProvider
}

// Publisher is the Framework boundary for tenant notification delivery.
type Publisher interface {
	Create(context.Context, CreateInput) (*Notification, error)
}

// CreateInput is a tenant-scoped notification command. The target member is
// optional; omitting it delegates delivery to the host's tenant policy.
type CreateInput struct {
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Type        string         `json:"type,omitempty"`
	Category    string         `json:"category,omitempty"`
	IsImportant bool           `json:"is_important,omitempty"`
	MemberUUID  string         `json:"member_uuid,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Notification struct {
	UUID        string         `json:"id"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Type        string         `json:"type"`
	Category    string         `json:"category"`
	IsRead      bool           `json:"isRead"`
	IsImportant bool           `json:"isImportant"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	MemberUUID  string         `json:"userId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx notifications request failed: status=%d", e.StatusCode)
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx notifications base_url is required")
	}
	c := newClient(cfg, httpClient)
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		c.tokens = staticToken(token)
		return c, nil
	}
	stsClient, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, fmt.Errorf("powerx notifications delegated sts configuration: %w", err)
	}
	c.tokens = stsToken{client: stsClient}
	return c, nil
}

func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx notifications base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx notifications token provider is required")
	}
	c := newClient(cfg, httpClient)
	c.tokens = provider
	return c, nil
}

func newClient(cfg Config, httpClient *http.Client) *Client {
	c := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")}
	if httpClient != nil {
		c.http = httpClient
		return c
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	c.http = &http.Client{Timeout: timeout}
	return c
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
	var out Notification
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/tenant/notifications", input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.UUID) == "" {
		return nil, errors.New("powerx notifications response missing notification uuid")
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
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("powerx notifications token provider returned an empty token")
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(raw)}
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return errors.New("powerx notifications response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

type stsToken struct{ client *sts.Client }

func (t stsToken) Token(ctx context.Context) (string, error) {
	if t.client == nil {
		return "", errors.New("powerx notifications sts client is required")
	}
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
