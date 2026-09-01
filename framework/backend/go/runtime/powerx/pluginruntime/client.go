// Package pluginruntime exposes tenant-scoped PowerX plugin runtime objects.
package pluginruntime

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

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type Config struct {
	BaseURL         string
	BearerToken     string
	STSClientID     string
	STSClientSecret string
	STSTokenURL     string
	Timeout         time.Duration
}

type KnowledgeSpace struct {
	UUID           string `json:"uuid"`
	SpaceName      string `json:"space_name"`
	Status         string `json:"status"`
	DepartmentCode string `json:"department_code"`
	RAGProfileKey  string `json:"rag_profile_key"`
	UpdatedAt      string `json:"updated_at"`
	CreatedAt      string `json:"created_at"`
}

type ListKnowledgeSpacesInput struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

type ListKnowledgeSpacesOutput struct {
	Items    []KnowledgeSpace
	Total    int64
	Page     int
	PageSize int
}

type Agent struct {
	UUID             string         `json:"uuid"`
	Key              string         `json:"key"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Persona          string         `json:"persona,omitempty"`
	Environment      string         `json:"env"`
	Status           string         `json:"status"`
	Visibility       string         `json:"visibility"`
	Scope            string         `json:"scope"`
	Source           string         `json:"source"`
	TypeID           string         `json:"type_id,omitempty"`
	Scene            string         `json:"scene,omitempty"`
	PromptSeed       string         `json:"prompt_seed,omitempty"`
	Parameters       map[string]any `json:"parameters,omitempty"`
	SkillIDs         []string       `json:"skill_ids,omitempty"`
	KnowledgeBaseIDs []string       `json:"knowledge_base_ids,omitempty"`
	Meta             map[string]any `json:"meta,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

type InstantiateAgentInput struct {
	Environment      string         `json:"env,omitempty"`
	Key              string         `json:"key,omitempty"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	TypeID           string         `json:"type_id,omitempty"`
	Scene            string         `json:"scene,omitempty"`
	PromptSeed       string         `json:"prompt_seed,omitempty"`
	Persona          string         `json:"persona,omitempty"`
	SkillIDs         []string       `json:"skill_ids,omitempty"`
	KnowledgeBaseIDs []string       `json:"knowledge_base_ids,omitempty"`
	Parameters       map[string]any `json:"parameters,omitempty"`
	Meta             map[string]any `json:"meta,omitempty"`
	Status           string         `json:"status,omitempty"`
	Visibility       string         `json:"visibility,omitempty"`
	Scope            string         `json:"scope,omitempty"`
}

type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenProvider
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx plugin runtime base_url is required")
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
		return nil, fmt.Errorf("powerx plugin runtime delegated sts configuration: %w", err)
	}
	c.tokens = stsToken{client: client}
	return c, nil
}

func (c *Client) ListKnowledgeSpaces(ctx context.Context, input ListKnowledgeSpacesInput) (*ListKnowledgeSpacesOutput, error) {
	query := url.Values{}
	if input.Page > 0 {
		query.Set("page", fmt.Sprint(input.Page))
	}
	if input.PageSize > 0 {
		query.Set("page_size", fmt.Sprint(input.PageSize))
	}
	if value := strings.TrimSpace(input.Status); value != "" {
		query.Set("status", value)
	}
	if value := strings.TrimSpace(input.Keyword); value != "" {
		query.Set("keyword", value)
	}
	path := "/api/v1/tenant/plugin-runtime/knowledge-spaces"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var raw struct {
		Items      []KnowledgeSpace `json:"items"`
		Pagination struct {
			Total    int64 `json:"total"`
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
		} `json:"pagination"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return &ListKnowledgeSpacesOutput{Items: raw.Items, Total: raw.Pagination.Total, Page: raw.Pagination.Page, PageSize: raw.Pagination.PageSize}, nil
}

func (c *Client) InstantiateAgent(ctx context.Context, input InstantiateAgentInput) (*Agent, error) {
	var out Agent
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/tenant/plugin-runtime/agents/instantiate", input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.UUID) == "" {
		return nil, errors.New("powerx plugin runtime response missing agent uuid")
	}
	return &out, nil
}

func (c *Client) ListAgents(ctx context.Context, environment, status string) ([]Agent, error) {
	query := url.Values{}
	if value := strings.TrimSpace(environment); value != "" {
		query.Set("env", value)
	}
	if value := strings.TrimSpace(status); value != "" {
		query.Set("status", value)
	}
	path := "/api/v1/tenant/plugin-runtime/agents"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var raw struct {
		Items []Agent `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Items, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, input, output any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("powerx plugin runtime client is not configured")
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
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
		return errors.New("powerx plugin runtime response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx plugin runtime request failed: status=%d", e.StatusCode)
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) {
	if strings.TrimSpace(string(t)) == "" {
		return "", errors.New("powerx plugin runtime bearer token is required")
	}
	return string(t), nil
}

type stsToken struct{ client *sts.Client }

func (t stsToken) Token(ctx context.Context) (string, error) {
	if t.client == nil {
		return "", errors.New("powerx plugin runtime sts client is required")
	}
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
