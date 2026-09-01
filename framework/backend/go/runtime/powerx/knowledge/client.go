// Package knowledge provides the typed delegated client for Core QA bridge APIs.
package knowledge

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
type tokenProvider interface {
	Token(context.Context) (string, error)
}
type TokenProvider = tokenProvider

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Client struct {
	baseURL string
	http    *http.Client
	token   tokenProvider
}

// QABridge is the Framework boundary for the currently published knowledge
// retrieval-plan and memory-snapshot operations.
type QABridge interface {
	RetrievalPlan(context.Context, RetrievalPlanInput) (*RetrievalPlan, error)
	UpsertMemorySnapshot(context.Context, MemorySnapshotInput) (*MemorySnapshot, error)
}

type RetrievalPlanInput struct {
	Intent          string   `json:"intent"`
	DomainTags      []string `json:"domainTags,omitempty"`
	SessionID       string   `json:"sessionId,omitempty"`
	LatencyBudgetMs int      `json:"latencyBudgetMs,omitempty"`
}
type PlanSpace struct {
	SpaceUUID        string  `json:"spaceId"`
	SpaceName        string  `json:"spaceName"`
	Strategy         string  `json:"strategy"`
	CitationCoverage float64 `json:"citationCoverage"`
	DegradeReason    string  `json:"degradeReason,omitempty"`
}
type RetrievalPlan struct {
	TenantUUID      string      `json:"tenant_uuid"`
	Intent          string      `json:"intent"`
	CandidateSpaces []PlanSpace `json:"candidateSpaces"`
	SessionID       string      `json:"sessionId"`
	TraceID         string      `json:"-"`
}
type MemoryCitation struct {
	ChunkID     string   `json:"chunkId"`
	SpaceUUID   string   `json:"spaceId"`
	Status      string   `json:"status"`
	Citations   []string `json:"citations"`
	SourceType  string   `json:"sourceType"`
	Confidence  float64  `json:"confidence"`
	DeltaReason string   `json:"deltaReason"`
}
type MemorySnapshotInput struct {
	SessionID string           `json:"sessionId"`
	Updates   []MemoryCitation `json:"updates,omitempty"`
	TraceID   string           `json:"traceId,omitempty"`
}
type MemorySnapshot struct {
	TenantUUID string           `json:"tenant_uuid"`
	SessionID  string           `json:"sessionId"`
	Citations  []MemoryCitation `json:"citations"`
	TraceID    string           `json:"traceId,omitempty"`
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx knowledge base_url is required")
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
		c.token = staticToken(token)
		return c, nil
	}
	client, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, fmt.Errorf("powerx knowledge delegated sts configuration: %w", err)
	}
	c.token = stsToken{client}
	return c, nil
}

// NewClientWithTokenProvider shares Skeleton's delegated STS flow instead of
// creating a second token configuration for the QA bridge.
func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx knowledge base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx knowledge token provider is required")
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
func (c *Client) RetrievalPlan(ctx context.Context, input RetrievalPlanInput) (*RetrievalPlan, error) {
	var out RetrievalPlan
	if err := c.do(ctx, "/api/v1/openapi/knowledge-spaces/qa/retrieval-plan", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) UpsertMemorySnapshot(ctx context.Context, input MemorySnapshotInput) (*MemorySnapshot, error) {
	var out MemorySnapshot
	if err := c.do(ctx, "/api/v1/openapi/knowledge-spaces/qa/memory-snapshot", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) do(ctx context.Context, path string, input, output any) error {
	if c == nil || c.http == nil || c.token == nil {
		return errors.New("powerx knowledge client is not configured")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
		return errors.New("powerx knowledge response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx knowledge request failed: status=%d", e.StatusCode)
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
