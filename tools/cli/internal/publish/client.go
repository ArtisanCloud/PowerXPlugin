package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	PluginID     string
	Version      string
	Channel      string
	Notes        string
	PackagePath  string
	MetadataPath string
	ManifestPath string
	RBACPath     string
	CLIVersion   string
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
	if req.PackagePath == "" {
		return nil, fmt.Errorf("package path is required")
	}
	if req.MetadataPath == "" {
		return nil, fmt.Errorf("metadata path is required")
	}
	if req.ManifestPath == "" {
		return nil, fmt.Errorf("manifest path is required")
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

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writeField := func(key, value string) error {
		if value == "" {
			return nil
		}
		return writer.WriteField(key, value)
	}

	if err := writeField("pluginId", req.PluginID); err != nil {
		return nil, err
	}
	if err := writeField("version", req.Version); err != nil {
		return nil, err
	}
	if err := writeField("channel", req.Channel); err != nil {
		return nil, err
	}
	if err := writeField("notes", req.Notes); err != nil {
		return nil, err
	}
	if err := writeField("cliVersion", req.CLIVersion); err != nil {
		return nil, err
	}

	if err := addFilePart(writer, "package", req.PackagePath); err != nil {
		return nil, err
	}
	if err := addFilePart(writer, "metadata", req.MetadataPath); err != nil {
		return nil, err
	}
	if err := addFilePart(writer, "manifest", req.ManifestPath); err != nil {
		return nil, err
	}
	if req.RBACPath != "" {
		if err := addFilePart(writer, "rbac", req.RBACPath); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := c.baseURL + "/internal/plugins/releases"
	if ctx == nil {
		ctx = context.Background()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create publish request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
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

	if envelope.Code != 0 && envelope.Code != 200 {
		if envelope.Message == "" {
			envelope.Message = "registry returned non-success code"
		}
		return nil, fmt.Errorf("publish failed (code %d): %s", envelope.Code, envelope.Message)
	}
	if envelope.Data.PublishID == "" {
		return nil, fmt.Errorf("publish response missing publishId")
	}

	return &SubmitResponse{
		PublishID: envelope.Data.PublishID,
		ReviewURL: envelope.Data.ReviewURL,
		Status:    envelope.Data.Status,
		Message:   envelope.Message,
	}, nil
}

func addFilePart(writer *multipart.Writer, fieldName, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("stream %s: %w", path, err)
	}
	return nil
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
		PublishID string `json:"publishId"`
		ReviewURL string `json:"reviewUrl"`
		Status    string `json:"status"`
	} `json:"data"`
}
