// Package pluginrelease provides the UUID-only tenant Plugin Release Host Contract.
package pluginrelease

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

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type TokenProvider interface {
	Token(context.Context) (string, error)
}
type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Config struct {
	BaseURL string
	Timeout time.Duration
}
type Client struct {
	baseURL string
	tokens  TokenProvider
	http    *http.Client
}

func NewClientWithTokenProvider(cfg Config, tokens TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("plugin release base_url is required")
	}
	if tokens == nil {
		return nil, errors.New("plugin release token provider is required")
	}
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), tokens: tokens, http: httpClient}, nil
}

type InstallSession struct {
	SessionUUID  string   `json:"session_uuid"`
	PluginID     string   `json:"plugin_id"`
	ServiceActor string   `json:"service_actor"`
	ArtifactURI  string   `json:"artifact_uri"`
	FeatureFlags []string `json:"feature_flags,omitempty"`
	Status       string   `json:"status"`
	LogURL       string   `json:"log_url,omitempty"`
	CreatedAt    string   `json:"created_at"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
}
type StartInstallSessionInput struct {
	ArtifactURI  string   `json:"artifact_uri"`
	FeatureFlags []string `json:"feature_flags,omitempty"`
	ResetCache   bool     `json:"reset_cache,omitempty"`
}
type StopInstallSessionInput struct {
	Force bool `json:"force,omitempty"`
}
type ImportJob struct {
	JobUUID     string `json:"job_uuid"`
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at,omitempty"`
}
type StartImportJobInput struct {
	PackageUUID     string `json:"package_uuid"`
	LicenseAccepted bool   `json:"license_accepted"`
	DryRun          bool   `json:"dry_run,omitempty"`
}

func (c *Client) StartInstallSession(ctx context.Context, in StartInstallSessionInput) (*InstallSession, error) {
	var out InstallSession
	if err := c.do(ctx, http.MethodPost, "/api/v1/tenant/plugin-release/install-sessions", in, &out); err != nil {
		return nil, err
	}
	if out.SessionUUID == "" {
		return nil, errors.New("plugin release response missing session_uuid")
	}
	return &out, nil
}
func (c *Client) GetInstallSession(ctx context.Context, sessionUUID string) (*InstallSession, error) {
	if strings.TrimSpace(sessionUUID) == "" {
		return nil, errors.New("session_uuid is required")
	}
	var out InstallSession
	if err := c.do(ctx, http.MethodGet, "/api/v1/tenant/plugin-release/install-sessions/"+url.PathEscape(sessionUUID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) StopInstallSession(ctx context.Context, sessionUUID string, in StopInstallSessionInput) (*InstallSession, error) {
	if strings.TrimSpace(sessionUUID) == "" {
		return nil, errors.New("session_uuid is required")
	}
	var out struct {
		SessionUUID string `json:"session_uuid"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/tenant/plugin-release/install-sessions/"+url.PathEscape(sessionUUID)+"/stop", in, &out); err != nil {
		return nil, err
	}
	if out.SessionUUID == "" || out.SessionUUID != sessionUUID {
		return nil, hostError(http.StatusBadGateway, nil)
	}
	// Core acknowledges the command without a lifecycle status. The caller
	// must read GetInstallSession for authoritative state.
	return &InstallSession{SessionUUID: out.SessionUUID}, nil
}
func (c *Client) StartImportJob(ctx context.Context, in StartImportJobInput) (*ImportJob, error) {
	if strings.TrimSpace(in.PackageUUID) == "" {
		return nil, errors.New("package_uuid is required")
	}
	var out ImportJob
	if err := c.do(ctx, http.MethodPost, "/api/v1/tenant/plugin-release/import-jobs", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) GetImportJob(ctx context.Context, jobUUID string) (*ImportJob, error) {
	if strings.TrimSpace(jobUUID) == "" {
		return nil, errors.New("job_uuid is required")
	}
	var out ImportJob
	if err := c.do(ctx, http.MethodGet, "/api/v1/tenant/plugin-release/import-jobs/"+url.PathEscape(jobUUID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) do(ctx context.Context, method, path string, input, out any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("plugin release client is not configured")
	}
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return hostError(http.StatusServiceUnavailable, nil)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return hostError(http.StatusServiceUnavailable, nil)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return hostError(http.StatusBadGateway, nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return hostError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Data) == 0 || string(bytes.TrimSpace(envelope.Data)) == "null" {
		return hostError(http.StatusBadGateway, nil)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return hostError(http.StatusBadGateway, nil)
	}
	switch value := out.(type) {
	case *InstallSession:
		if strings.TrimSpace(value.SessionUUID) == "" {
			return hostError(http.StatusBadGateway, nil)
		}
	case *ImportJob:
		if strings.TrimSpace(value.JobUUID) == "" {
			return hostError(http.StatusBadGateway, nil)
		}
	}
	return nil
}

type HTTPError struct {
	StatusCode int
	ReasonCode string
	Body       string
}

func (e *HTTPError) Error() string {
	if e != nil && e.ReasonCode != "" {
		return fmt.Sprintf("plugin release request failed: reason=%s status=%d", e.ReasonCode, e.StatusCode)
	}
	return fmt.Sprintf("plugin release request failed: status=%d", e.StatusCode)
}

func hostError(status int, raw []byte) *HTTPError {
	reason := hostcontract.ParseReasonCode(raw, "")
	if reason == "" {
		switch status {
		case http.StatusBadRequest:
			reason = "PLUGIN_RELEASE_INVALID_ARGUMENT"
		case http.StatusUnauthorized:
			reason = "PLUGIN_RELEASE_UNAUTHORIZED"
		case http.StatusForbidden:
			reason = "PLUGIN_RELEASE_FORBIDDEN"
		case http.StatusNotFound:
			reason = "PLUGIN_RELEASE_SESSION_NOT_FOUND"
		case http.StatusConflict:
			reason = "PLUGIN_RELEASE_ACTIVE_SESSION_CONFLICT"
		case http.StatusUnprocessableEntity:
			reason = "PLUGIN_RELEASE_PACKAGE_VERIFICATION_FAILED"
		default:
			reason = "PLUGIN_RELEASE_UPSTREAM_DEPENDENCY"
		}
	}
	return &HTTPError{StatusCode: status, ReasonCode: reason, Body: string(raw)}
}
