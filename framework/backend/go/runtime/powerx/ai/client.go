package ai

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

const defaultTimeout = 5 * time.Minute

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Config struct {
	BaseURL string
	// AuthScheme is either "bearer" (the default) or "apikey".  API-key
	// authentication is required by local PowerX gateway deployments.
	AuthScheme      string
	BearerToken     string
	APIKey          string
	STSClientID     string
	STSClientSecret string
	STSTokenURL     string
	Timeout         time.Duration
}

type Client struct {
	baseURL       string
	http          *http.Client
	authorization authorizationProvider
}

type authorizationProvider interface {
	Authorization(context.Context) (string, error)
}

type authorizationProviderFunc func(context.Context) (string, error)

func (f authorizationProviderFunc) Authorization(ctx context.Context) (string, error) { return f(ctx) }

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx ai base_url is required")
	}
	client := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")}
	if httpClient != nil {
		client.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		client.http = &http.Client{Timeout: timeout}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.AuthScheme), "apikey") || strings.EqualFold(strings.TrimSpace(cfg.AuthScheme), "api_key") || strings.EqualFold(strings.TrimSpace(cfg.AuthScheme), "api-key") {
		if key := strings.TrimSpace(cfg.APIKey); key != "" {
			client.authorization = staticAuthorization("ApiKey " + key)
			return client, nil
		}
		return nil, errors.New("powerx ai api key is required")
	}
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		client.authorization = bearerToken(token)
		return client, nil
	}
	stsClient, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, fmt.Errorf("powerx ai delegated sts configuration: %w", err)
	}
	client.authorization = stsAuthorization{client: stsClient}
	return client, nil
}

// NewClientWithTokenProvider reuses the plugin-owned short-lived STS provider.
func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" || provider == nil {
		return nil, errors.New("powerx ai base_url and token provider are required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), http: httpClient, authorization: bearerTokenProvider{provider: provider}}, nil
}

func (c *Client) LLMInvoke(ctx context.Context, input LLMInvokeInput) (*LLMInvokeOutput, error) {
	var raw struct {
		Output struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"output"`
		Meta  map[string]any `json:"meta"`
		Usage map[string]any `json:"usage"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/ai/llm/invoke", input, &raw); err != nil {
		return nil, err
	}
	return &LLMInvokeOutput{Type: raw.Output.Type, Text: raw.Output.Text, FinishReason: stringValue(raw.Meta, "finish_reason"), Usage: raw.Usage}, nil
}

func (c *Client) ListLLMModels(ctx context.Context, provider string) (*ListLLMModelsOutput, error) {
	path := "/api/v1/ai/llm/models"
	if provider = strings.TrimSpace(provider); provider != "" {
		path += "?" + url.Values{"provider": []string{provider}}.Encode()
	}
	var out ListLLMModelsOutput
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateLLMSession(ctx context.Context, input CreateLLMSessionInput) (*LLMSession, error) {
	var out LLMSession
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/ai/llm/sessions", input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.SessionID) == "" {
		return nil, errors.New("powerx ai response missing session_id")
	}
	return &out, nil
}

func (c *Client) AppendLLMSessionMessage(ctx context.Context, sessionID string, input AppendLLMSessionMessageInput) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("powerx ai session_id is required")
	}
	return c.doJSON(ctx, http.MethodPost, "/api/v1/ai/llm/sessions/"+url.PathEscape(strings.TrimSpace(sessionID))+"/messages", input, nil)
}

func (c *Client) EmbeddingInvoke(ctx context.Context, input EmbeddingInvokeInput) (*EmbeddingInvokeOutput, error) {
	var out EmbeddingInvokeOutput
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/ai/embedding/invoke", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) VLMInvoke(ctx context.Context, input ModalInvokeInput) (*ModalInvokeOutput, error) {
	return c.modalInvoke(ctx, "/api/v1/ai/vlm/invoke", input)
}

func (c *Client) ImageInvoke(ctx context.Context, input ModalInvokeInput) (*ModalInvokeOutput, error) {
	return c.modalInvoke(ctx, "/api/v1/ai/image/invoke", input)
}

func (c *Client) VideoInvoke(ctx context.Context, input ModalInvokeInput) (*ModalInvokeOutput, error) {
	return c.modalInvoke(ctx, "/api/v1/ai/video/invoke", input)
}

func (c *Client) TTSInvoke(ctx context.Context, input ModalInvokeInput) (*ModalInvokeOutput, error) {
	return c.modalInvoke(ctx, "/api/v1/ai/tts/invoke", input)
}

func (c *Client) modalInvoke(ctx context.Context, path string, input ModalInvokeInput) (*ModalInvokeOutput, error) {
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, path, input, &raw); err != nil {
		return nil, err
	}
	return &ModalInvokeOutput{Data: append(json.RawMessage(nil), raw...)}, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, input any, output any) error {
	if c == nil || c.http == nil || c.authorization == nil {
		return errors.New("powerx ai client is not configured")
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
	authorization, err := c.authorization.Authorization(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
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
		return errors.New("powerx ai response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx ai request failed: status=%d", e.StatusCode)
}

type staticAuthorization string

func (a staticAuthorization) Authorization(context.Context) (string, error) {
	if strings.TrimSpace(string(a)) == "" {
		return "", errors.New("powerx ai authorization is required")
	}
	return string(a), nil
}

type bearerToken string

func (t bearerToken) Authorization(context.Context) (string, error) {
	if strings.TrimSpace(string(t)) == "" {
		return "", errors.New("powerx ai bearer token is required")
	}
	return "Bearer " + string(t), nil
}

type bearerTokenProvider struct{ provider TokenProvider }

func (p bearerTokenProvider) Authorization(ctx context.Context) (string, error) {
	if p.provider == nil {
		return "", errors.New("powerx ai token provider is required")
	}
	token, err := p.provider.Token(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", errors.New("powerx ai bearer token is required")
	}
	return "Bearer " + token, nil
}

type stsAuthorization struct{ client *sts.Client }

func (t stsAuthorization) Authorization(ctx context.Context) (string, error) {
	if t.client == nil {
		return "", errors.New("powerx ai sts client is required")
	}
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("powerx ai sts token is empty")
	}
	return "Bearer " + token.AccessToken, nil
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}
