package runtimeexample

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
)

func localConnectionError(status int, reason string, upstream int) error {
	details := map[string]any{"generation_submitted": false, "api_key_verified": false}
	if upstream != 0 {
		details["upstream_status"] = upstream
	}
	return &powerxai.HTTPError{StatusCode: status, ErrorCode: reason, ReasonCode: reason, Details: details}
}

// ProbeProviderConnection implements the local adapter's read-only Ollama
// probe. It never resolves a Core model, sends a prompt or persists a profile.
func (s *LocalAI) ProbeProviderConnection(ctx context.Context, provider, endpoint string) (*fw.ConnectionVerification, error) {
	if provider != "ollama" {
		return nil, localConnectionError(501, "AI_CONNECTION_PROBE_UNSUPPORTED", 0)
	}
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(endpoint), "/"))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, localConnectionError(422, "AI_CONNECTION_CONFIG_INVALID", 0)
	}
	if s == nil || s.http == nil {
		return nil, localConnectionError(503, "AI_CLIENT_UNAVAILABLE", 0)
	}
	u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/v1"), "/api") + "/api/tags"
	u.RawPath = ""
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, localConnectionError(422, "AI_CONNECTION_CONFIG_INVALID", 0)
	}
	req.Header.Set("Accept", "application/json")
	client := *s.http
	client.Timeout = 10 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, localConnectionError(504, "AI_CONNECTION_TIMEOUT", 0)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, localConnectionError(502, "AI_CONNECTION_UNREACHABLE", 0)
	}
	defer response.Body.Close()
	switch status := response.StatusCode; {
	case status == 401 || status == 403:
		return nil, localConnectionError(422, "AI_CONNECTION_AUTH_FAILED", status)
	case status == 404 || status == 405 || status == 501:
		return nil, localConnectionError(501, "AI_CONNECTION_PROBE_UNSUPPORTED", status)
	case status == 429:
		return nil, localConnectionError(429, "AI_CONNECTION_RATE_LIMITED", status)
	case status >= 300 && status < 400:
		return nil, localConnectionError(502, "AI_CONNECTION_REDIRECT_REJECTED", status)
	case status < 200 || status >= 300:
		return nil, localConnectionError(502, "AI_CONNECTION_UPSTREAM_FAILED", status)
	}
	const maxResponse = 2 << 20
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, localConnectionError(504, "AI_CONNECTION_TIMEOUT", response.StatusCode)
	}
	var body struct {
		Models []json.RawMessage `json:"models"`
	}
	if err != nil || len(raw) > maxResponse || json.Unmarshal(raw, &body) != nil || body.Models == nil {
		return nil, localConnectionError(502, "AI_CONNECTION_RESPONSE_INVALID", response.StatusCode)
	}
	return &fw.ConnectionVerification{Provider: provider, Probe: "ollama_tags", EndpointReachable: true, Authentication: "not_required"}, nil
}
