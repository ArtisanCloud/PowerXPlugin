package customerfw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type DelegatedClientConfig struct {
	BaseURL string
	Timeout time.Duration
	Client  *http.Client
}

type DelegatedCustomerAuthClient struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewDelegatedCustomerAuthClient(cfg DelegatedClientConfig) *DelegatedCustomerAuthClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 500*time.Millisecond}
	}
	return &DelegatedCustomerAuthClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		client:  client,
		timeout: timeout,
	}
}

func (c *DelegatedCustomerAuthClient) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	var out AuthResult
	if err := c.post(ctx, "/customer/auth/register", input, &out); err != nil {
		return nil, err
	}
	out.Context = NormalizeContext(out.Context)
	return &out, nil
}

func (c *DelegatedCustomerAuthClient) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	var out AuthResult
	path := "/customer/auth/login"
	if IsWeChatMiniAppLogin(input) {
		path = "/customer/auth/wechat/login"
	}
	if err := c.post(ctx, path, input, &out); err != nil {
		return nil, err
	}
	out.Context = NormalizeContext(out.Context)
	return &out, nil
}

func (c *DelegatedCustomerAuthClient) Validate(ctx context.Context, token string) (*CustomerContext, error) {
	var out struct {
		Context *CustomerContext `json:"context"`
	}
	if err := c.post(ctx, "/customer/auth/validate", map[string]string{"token": token}, &out); err != nil {
		return nil, err
	}
	return NormalizeContext(out.Context), nil
}

func (c *DelegatedCustomerAuthClient) post(ctx context.Context, path string, payload any, out any) error {
	if c == nil || strings.TrimSpace(c.baseURL) == "" {
		return NewError(CodeCustomerDelegateUnavailable, "customer auth delegate unavailable")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer auth delegate unavailable", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return NewError(CodeCustomerTokenInvalid, "customer token invalid")
	}
	if resp.StatusCode >= 500 {
		return NewError(CodeCustomerDelegateUnavailable, "customer auth delegate unavailable")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NewError(CodeCustomerTokenInvalid, "customer auth failed")
	}
	return decodeEnvelope(raw, out)
}

func decodeEnvelope(raw []byte, out any) error {
	if len(raw) == 0 {
		return errors.New("empty response")
	}
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Success != nil {
		if !*envelope.Success {
			return NewError(CodeCustomerTokenInvalid, "customer auth failed")
		}
		if len(envelope.Data) == 0 {
			return errors.New("missing data")
		}
		return json.Unmarshal(envelope.Data, out)
	}
	return json.Unmarshal(raw, out)
}
