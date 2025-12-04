package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Options configures the publish client.
type Options struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Client uploads plugin packages to a PowerX Registry.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// SubmitRequest describes a publish submission.
type SubmitRequest struct {
	TenantUUID      string            `json:"tenantUuid"`
	PluginID        string            `json:"pluginId"`
	Version         string            `json:"version"`
	Channel         string            `json:"channel"`
	ReleaseNotes    string            `json:"releaseNotes,omitempty"`
	BuildArtifact   string            `json:"buildArtifactUri"`
	CommitHash      string            `json:"commitHash,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ApprovalContext string            `json:"approvalContext,omitempty"`
	CLIVersion      string            `json:"cliVersion,omitempty"`
}

// SubmitResponse contains registry response details.
type SubmitResponse struct {
	PublishID string
	ReviewURL string
	Status    string
	Message   string
}

// NewClient constructs a publish client with sane defaults.
func NewClient(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("publish base URL is required")
	}
	base := strings.TrimSuffix(opts.BaseURL, "/")
	httpClient := opts.HTTPClient
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}
	return &Client{
		baseURL:    base,
		apiToken:   opts.APIToken,
		httpClient: httpClient,
	}, nil
}

// Submit uploads the package + metadata to the registry.
func (c *Client) Submit(ctx context.Context, req *SubmitRequest) (*SubmitResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is required")
	}
	if req.TenantUUID == "" {
		return nil, fmt.Errorf("tenant UUID is required")
	}
	if req.PluginID == "" {
		return nil, fmt.Errorf("plugin ID is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("plugin version is required")
	}
	if req.Channel == "" {
		return nil, fmt.Errorf("channel is required (use --channel)")
	}
	if req.BuildArtifact == "" {
		return nil, fmt.Errorf("buildArtifactUri is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode publish payload: %w", err)
	}

	endpoint := c.baseURL + "/internal/plugins/releases"
	if ctx == nil {
		ctx = context.Background()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create publish request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if ua := buildUserAgent(req.CLIVersion); ua != "" {
		httpReq.Header.Set("User-Agent", ua)
	}
	if c.apiToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("publish request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read publish response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("publish failed (HTTP %d): %s", resp.StatusCode, truncate(respBody, 512))
	}

	var envelope registryResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("decode publish response: %w", err)
	}

	if envelope.Code != 0 && envelope.Code != 200 && envelope.Code != 201 {
		if envelope.Message == "" {
			envelope.Message = "registry returned non-success code"
		}
		return nil, fmt.Errorf("publish failed (code %d): %s", envelope.Code, envelope.Message)
	}

	pubID := envelope.Data.PublishID
	if pubID == "" {
		pubID = envelope.Data.CandidateID
	}
	if pubID == "" {
		return nil, fmt.Errorf("publish response missing publishId/candidateId")
	}

	return &SubmitResponse{
		PublishID: pubID,
		ReviewURL: envelope.Data.ReviewURL,
		Status:    envelope.Data.Status,
		Message:   envelope.Message,
	}, nil
}

func buildUserAgent(version string) string {
	if version == "" {
		return "px-plugin/dev"
	}
	return fmt.Sprintf("px-plugin/%s", version)
}

func truncate(b []byte, max int) string {
	s := string(bytes.TrimSpace(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type registryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PublishID   string `json:"publishId"`
		CandidateID string `json:"candidateId"`
		ReviewURL   string `json:"reviewUrl"`
		Status      string `json:"status"`
	} `json:"data"`
}
