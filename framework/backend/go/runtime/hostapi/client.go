// Package hostapi supplies credential-scoped transport for Runtime Host v1.
package hostapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// Credential must be supplied by the trusted STS exchange/bootstrap layer.
// TenantUUID describes the very same token, never caller-provided JSON.
type Credential struct {
	Token      string
	TenantUUID string
}
type TokenProvider interface {
	Credential(context.Context) (Credential, error)
}
type TokenProviderFunc func(context.Context) (Credential, error)

func (f TokenProviderFunc) Credential(ctx context.Context) (Credential, error) { return f(ctx) }

type Config struct {
	BaseURL string
	Timeout time.Duration
}
type HTTPError struct {
	StatusCode int
	ReasonCode string
	RequestID  string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("%s: HTTP %d", e.ReasonCode, e.StatusCode) }

type Client struct {
	base   string
	tokens TokenProvider
	http   *http.Client
	prefix string
}

func New(cfg Config, tokens TokenProvider, h *http.Client, prefix string) (*Client, error) {
	if tokens != nil {
		v := reflect.ValueOf(tokens)
		if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Func || v.Kind() == reflect.Interface) && v.IsNil() {
			tokens = nil
		}
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || tokens == nil {
		return nil, fmt.Errorf("%s_INVALID_ARGUMENT", prefix)
	}
	if h == nil {
		h = &http.Client{Timeout: cfg.Timeout}
		if h.Timeout <= 0 {
			h.Timeout = 30 * time.Second
		}
	}
	clone := *h
	clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{strings.TrimRight(cfg.BaseURL, "/"), tokens, &clone, prefix}, nil
}
func (c *Client) Failure(status int, suffix string) *HTTPError {
	return &HTTPError{StatusCode: status, ReasonCode: c.prefix + "_" + suffix}
}
func (c *Client) Do(ctx context.Context, tenant, method, path string, in, out any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return &HTTPError{StatusCode: 503, ReasonCode: "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE"}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	credential, err := c.tokens.Credential(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || strings.TrimSpace(credential.Token) == "" {
		return c.Failure(503, "UPSTREAM_DEPENDENCY")
	}
	if credential.TenantUUID == "" || credential.TenantUUID != tenant {
		return c.Failure(403, "FORBIDDEN")
	}
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return c.Failure(400, "INVALID_ARGUMENT")
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return c.Failure(400, "INVALID_ARGUMENT")
	}
	req.Header.Set("Authorization", "Bearer "+credential.Token)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if ctx.Err() != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return ctx.Err()
	}
	if err != nil {
		return c.Failure(503, "UPSTREAM_DEPENDENCY")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || len(raw) > 2<<20 {
		return c.Failure(502, "INVALID_RESPONSE")
	}
	var env struct {
		Code      int             `json:"code"`
		Data      json.RawMessage `json:"data"`
		Reason    string          `json:"reason_code"`
		Error     string          `json:"error_code"`
		RequestID string          `json:"request_id"`
	}
	decodeErr := json.Unmarshal(raw, &env)
	if resp.StatusCode != 200 {
		reason := env.Reason
		if reason == "" {
			reason = env.Error
		}
		if reason == "" {
			reason = c.prefix + "_UPSTREAM_DEPENDENCY"
		}
		return &HTTPError{resp.StatusCode, reason, env.RequestID}
	}
	if decodeErr != nil || env.Code != 200 || len(env.Data) == 0 || bytes.Equal(bytes.TrimSpace(env.Data), []byte("null")) {
		return c.Failure(502, "INVALID_RESPONSE")
	}
	if out == nil {
		if string(bytes.TrimSpace(env.Data)) != "{}" {
			var m map[string]any
			if json.Unmarshal(env.Data, &m) != nil || len(m) != 0 {
				return c.Failure(502, "INVALID_RESPONSE")
			}
		}
		return nil
	}
	if json.Unmarshal(env.Data, out) != nil {
		return c.Failure(502, "INVALID_RESPONSE")
	}
	return nil
}
