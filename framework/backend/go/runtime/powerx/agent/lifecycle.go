package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HealthSummary is the tenant-scoped lifecycle health view returned by Core.
type HealthSummary struct {
	Status            string `json:"status"`
	HealthScore       int32  `json:"health_score"`
	UpdatedAt         string `json:"updated_at"`
	WindowDurationSec int32  `json:"window_duration_sec"`
	Metrics           struct {
		ThroughputPerMin float64 `json:"throughput_per_min"`
		SuccessRate      float64 `json:"success_rate"`
		P95LatencyMs     int32   `json:"p95_latency_ms"`
		ResourceUtilPct  float64 `json:"resource_util_pct"`
		ErrorRate        float64 `json:"error_rate"`
	} `json:"metrics"`
	Recommendations []string `json:"recommendations,omitempty"`
	AnomalyTraceIDs []string `json:"anomaly_trace_ids,omitempty"`
}

type HealthHistory struct {
	Snapshots []HealthSummary `json:"snapshots"`
}

type BridgeControlInput struct {
	Reason  string `json:"reason,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}
type BridgeRebalanceInput struct {
	TargetCapacityInstances int32  `json:"target_capacity_instances"`
	Reason                  string `json:"reason,omitempty"`
	TraceID                 string `json:"trace_id,omitempty"`
}

type BridgeAgent struct {
	UUID       string `json:"id"`
	TenantUUID string `json:"tenant_uuid"`
	Alias      string `json:"alias"`
	Status     string `json:"status"`
}
type BridgeLifecycleResult struct {
	Agent BridgeAgent     `json:"agent"`
	Event json.RawMessage `json:"event,omitempty"`
}

func (c *Client) GetHealthSummary(ctx context.Context, agentUUID string) (*HealthSummary, error) {
	var out HealthSummary
	if err := c.lifecycleJSON(ctx, http.MethodGet, "/api/v1/openapi/agents/"+url.PathEscape(strings.TrimSpace(agentUUID))+"/health/summary", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListHealthHistory(ctx context.Context, agentUUID string, rangeHours, limit int) (*HealthHistory, error) {
	query := url.Values{}
	if rangeHours > 0 {
		query.Set("range_hours", strconv.Itoa(rangeHours))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/v1/openapi/agents/" + url.PathEscape(strings.TrimSpace(agentUUID)) + "/health/history"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out HealthHistory
	if err := c.lifecycleJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBridgeState(ctx context.Context, agentUUID string, limit int) (*json.RawMessage, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/v1/openapi/agents/" + url.PathEscape(strings.TrimSpace(agentUUID)) + "/bridge/state"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out json.RawMessage
	if err := c.lifecycleJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Freeze(ctx context.Context, agentUUID string, input BridgeControlInput) (*BridgeLifecycleResult, error) {
	return c.control(ctx, agentUUID, "freeze", input)
}
func (c *Client) Recover(ctx context.Context, agentUUID string, input BridgeControlInput) (*BridgeLifecycleResult, error) {
	return c.control(ctx, agentUUID, "recover", input)
}

func (c *Client) Rebalance(ctx context.Context, agentUUID string, input BridgeRebalanceInput) (*BridgeLifecycleResult, error) {
	var out BridgeLifecycleResult
	if err := c.lifecycleJSON(ctx, http.MethodPost, "/api/v1/openapi/agents/"+url.PathEscape(strings.TrimSpace(agentUUID))+"/bridge/rebalance", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) control(ctx context.Context, agentUUID, action string, input BridgeControlInput) (*BridgeLifecycleResult, error) {
	var out BridgeLifecycleResult
	if err := c.lifecycleJSON(ctx, http.MethodPost, "/api/v1/openapi/agents/"+url.PathEscape(strings.TrimSpace(agentUUID))+"/bridge/"+action, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) lifecycleJSON(ctx context.Context, method, path string, input, output any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("agent lifecycle client is not configured")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err := c.authorize(ctx, req); err != nil {
		return err
	}
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
		return transportError(resp)
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
		return errors.New("agent lifecycle response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}
