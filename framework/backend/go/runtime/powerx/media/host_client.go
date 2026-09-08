package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

// HostTokenProvider obtains the shared plugin STS credential.
type HostTokenProvider interface {
	Token(context.Context) (string, error)
}
type HostTokenProviderFunc func(context.Context) (string, error)

func (f HostTokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type HostClientConfig struct {
	BaseURL string
	Timeout time.Duration
}
type HostClient struct {
	baseURL string
	tokens  HostTokenProvider
	http    *http.Client
}

func NewHostClientWithTokenProvider(cfg HostClientConfig, tokens HostTokenProvider, httpClient *http.Client) (*HostClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx media base_url is required")
	}
	if tokens == nil {
		return nil, errors.New("powerx media token provider is required")
	}
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &HostClient{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), tokens: tokens, http: httpClient}, nil
}

type HostAsset struct {
	AssetUUID        string `json:"asset_uuid"`
	Name             string `json:"name"`
	SizeBytes        int64  `json:"size_bytes"`
	MimeType         string `json:"mime_type"`
	OwnerSubjectUUID string `json:"owner_subject_uuid,omitempty"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}
type CreateAssetInput struct {
	Name             string `json:"name"`
	MimeType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
	Checksum         string `json:"checksum"`
	OwnerSubjectUUID string `json:"owner_subject_uuid,omitempty"`
}
type UpdateAssetInput struct {
	Name *string  `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}
type TransferTicket struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	ExpiresAt string            `json:"expires_at"`
	Headers   map[string]string `json:"headers,omitempty"`
}
type CompleteUploadInput struct {
	Checksum string `json:"checksum"`
}
type Variant struct {
	VariantUUID string `json:"variant_uuid"`
	AssetUUID   string `json:"asset_uuid"`
	Name        string `json:"name,omitempty"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
}
type CreateVariantInput struct {
	VariantType string `json:"variant_type"`
	Name        string `json:"name,omitempty"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

func (c *HostClient) ListAssets(ctx context.Context, in ListAssetsInput) (*ListAssetsOutput, error) {
	q := url.Values{}
	if in.Page > 0 {
		q.Set("page", strconv.Itoa(in.Page))
	}
	if in.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(in.PageSize))
	}
	if strings.TrimSpace(in.Keyword) != "" {
		q.Set("keyword", in.Keyword)
	}
	path := "/api/v1/tenant/media/assets"
	if q.Encode() != "" {
		path += "?" + q.Encode()
	}
	var raw struct {
		Items    []HostAsset `json:"items"`
		Total    int64       `json:"total"`
		Page     int         `json:"page"`
		PageSize int         `json:"page_size"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	items := make([]Asset, 0, len(raw.Items))
	for _, item := range raw.Items {
		items = append(items, Asset{UUID: item.AssetUUID, Name: item.Name, MimeType: item.MimeType})
	}
	return &ListAssetsOutput{Items: items, Total: raw.Total, Page: raw.Page, PageSize: raw.PageSize}, nil
}
func (c *HostClient) GetAsset(ctx context.Context, id string) (*HostAsset, error) {
	var out HostAsset
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/media/assets/"+url.PathEscape(id), nil, &out)
	return &out, err
}
func (c *HostClient) CreateAsset(ctx context.Context, in CreateAssetInput) (*HostAsset, error) {
	var out HostAsset
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/media/assets", in, &out)
	return &out, err
}
func (c *HostClient) UpdateAsset(ctx context.Context, id string, in UpdateAssetInput) (*HostAsset, error) {
	var out HostAsset
	err := c.do(ctx, http.MethodPatch, "/api/v1/tenant/media/assets/"+url.PathEscape(id), in, &out)
	return &out, err
}
func (c *HostClient) DeleteAsset(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/tenant/media/assets/"+url.PathEscape(id), nil, nil)
}
func (c *HostClient) PresignUpload(ctx context.Context, id string) (*TransferTicket, error) {
	var out TransferTicket
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/media/assets/"+url.PathEscape(id)+"/presign-upload", nil, &out)
	return &out, err
}
func (c *HostClient) CompleteUpload(ctx context.Context, id string, in CompleteUploadInput) (*HostAsset, error) {
	var out HostAsset
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/media/assets/"+url.PathEscape(id)+"/complete-upload", in, &out)
	return &out, err
}
func (c *HostClient) PresignDownload(ctx context.Context, id string) (*TransferTicket, error) {
	var out TransferTicket
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/media/assets/"+url.PathEscape(id)+"/presign-download", nil, &out)
	return &out, err
}
func (c *HostClient) CreateVariant(ctx context.Context, id string, in CreateVariantInput) (*Variant, error) {
	var out Variant
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/media/assets/"+url.PathEscape(id)+"/variants", in, &out)
	return &out, err
}
func (c *HostClient) GetVariant(ctx context.Context, id string) (*Variant, error) {
	var out Variant
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/media/assets/variants/"+url.PathEscape(id), nil, &out)
	return &out, err
}
func (c *HostClient) do(ctx context.Context, method, path string, input, out any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("powerx media Host client is not configured")
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
		return mediaHostError(http.StatusServiceUnavailable, nil)
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
		return mediaHostError(http.StatusServiceUnavailable, nil)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return mediaHostError(http.StatusBadGateway, nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mediaHostError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Data) == 0 || string(bytes.TrimSpace(envelope.Data)) == "null" {
		return mediaHostError(http.StatusBadGateway, nil)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return mediaHostError(http.StatusBadGateway, nil)
	}
	switch value := out.(type) {
	case *HostAsset:
		if strings.TrimSpace(value.AssetUUID) == "" {
			return mediaHostError(http.StatusBadGateway, nil)
		}
	case *Variant:
		if strings.TrimSpace(value.VariantUUID) == "" || strings.TrimSpace(value.AssetUUID) == "" {
			return mediaHostError(http.StatusBadGateway, nil)
		}
	case *TransferTicket:
		if strings.TrimSpace(value.URL) == "" || strings.TrimSpace(value.Method) == "" {
			return mediaHostError(http.StatusBadGateway, nil)
		}
	}
	return nil
}

type HostHTTPError struct {
	StatusCode int
	ReasonCode string
	Body       string
}

func (e *HostHTTPError) Error() string {
	if e != nil && e.ReasonCode != "" {
		return fmt.Sprintf("powerx media request failed: reason=%s status=%d", e.ReasonCode, e.StatusCode)
	}
	return fmt.Sprintf("powerx media request failed: status=%d", e.StatusCode)
}

func mediaHostError(status int, raw []byte) *HostHTTPError {
	reason := hostcontract.ParseReasonCode(raw, "")
	if reason == "" {
		switch status {
		case http.StatusBadRequest:
			reason = "MEDIA_INVALID_ARGUMENT"
		case http.StatusUnauthorized:
			reason = "MEDIA_UNAUTHORIZED"
		case http.StatusForbidden:
			reason = "MEDIA_FORBIDDEN"
		case http.StatusNotFound:
			reason = "MEDIA_ASSET_NOT_FOUND"
		default:
			reason = "MEDIA_UPSTREAM_DEPENDENCY"
		}
	}
	return &HostHTTPError{StatusCode: status, ReasonCode: reason, Body: string(raw)}
}
