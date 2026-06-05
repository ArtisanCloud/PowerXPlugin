package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const (
	CapabilityMediaAssetsRead   = "com.corex.media.assets.read"
	CapabilityMediaAssetsManage = "com.corex.media.assets.manage"
)

type GatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
}

type Client struct {
	gateway GatewayInvoker
	http    *http.Client
}

func NewClient(gatewayInvoker GatewayInvoker, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{gateway: gatewayInvoker, http: httpClient}
}

type UploadChannel string

const (
	UploadChannelPresigned UploadChannel = "presign_upload"
	UploadChannelDirect    UploadChannel = "direct_upload"
	UploadChannelExternal  UploadChannel = "external_link"
)

type BusinessStatus string

const (
	BusinessStatusDraft       BusinessStatus = "draft"
	BusinessStatusUnderReview BusinessStatus = "under_review"
	BusinessStatusPublished   BusinessStatus = "published"
	BusinessStatusArchived    BusinessStatus = "archived"
)

type PresignAction string

const (
	PresignActionUpload   PresignAction = "upload"
	PresignActionDownload PresignAction = "download"
)

type Asset struct {
	UUID             string            `json:"uuid"`
	TenantUUID       string            `json:"tenant_uuid"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Driver           string            `json:"driver"`
	Folder           string            `json:"folder"`
	ObjectKey        string            `json:"object_key"`
	ExternalURL      string            `json:"external_url"`
	SizeBytes        int64             `json:"size_bytes"`
	MimeType         string            `json:"mime_type"`
	OwnerSubjectType string            `json:"owner_subject_type"`
	OwnerSubjectID   string            `json:"owner_subject_id"`
	Tags             []string          `json:"tags"`
	BusinessStatus   BusinessStatus    `json:"business_status"`
	DownloadURL      string            `json:"download_url"`
	Extra            map[string]string `json:"extra"`
}

type AssetVariant struct {
	UUID        string            `json:"uuid"`
	TenantUUID  string            `json:"tenant_uuid"`
	AssetUUID   string            `json:"asset_uuid"`
	Variant     string            `json:"variant"`
	Name        string            `json:"name"`
	Driver      string            `json:"driver"`
	ObjectKey   string            `json:"object_key"`
	SizeBytes   int64             `json:"size_bytes"`
	MimeType    string            `json:"mime_type"`
	DownloadURL string            `json:"download_url"`
	Extra       map[string]string `json:"extra"`
}

type CreateAssetInput struct {
	TenantUUID       string
	OperatorID       string
	Name             string
	Description      string
	Driver           string
	Folder           string
	OwnerSubjectType string
	OwnerSubjectID   string
	Tags             []string
	UploadChannel    UploadChannel
	ExternalURL      string
	RequestID        string
}

type UpdateAssetInput struct {
	TenantUUID     string
	UUID           string
	OperatorID     string
	Name           *string
	Description    *string
	Tags           []string
	BusinessStatus *BusinessStatus
	RequestID      string
}

type GetAssetInput struct {
	TenantUUID string
	UUID       string
	RequestID  string
}

type PresignAssetInput struct {
	TenantUUID       string
	UUID             string
	OperatorID       string
	Action           PresignAction
	Method           string
	ExpiresInSeconds uint32
	Headers          map[string]string
	RequestID        string
}

type CreateAssetVariantInput struct {
	TenantUUID string
	UUID       string
	Variant    string
	Name       string
	Driver     string
	ObjectKey  string
	SizeBytes  int64
	MimeType   string
	Metadata   map[string]string
	RequestID  string
}

type PresignAssetVariantInput struct {
	TenantUUID       string
	UUID             string
	Variant          string
	OperatorID       string
	Action           PresignAction
	Method           string
	ExpiresInSeconds uint32
	Headers          map[string]string
	RequestID        string
}

type PresignAssetOutput struct {
	URL              string            `json:"url"`
	Method           string            `json:"method"`
	ExpiresInSeconds uint32            `json:"expires_in_seconds"`
	Headers          map[string]string `json:"headers"`
	Fields           map[string]string `json:"fields"`
	ObjectKey        string            `json:"object_key"`
}

func (c *Client) CreateAsset(ctx context.Context, input CreateAssetInput) (*Asset, error) {
	payload := map[string]any{
		"tenant_uuid":        strings.TrimSpace(input.TenantUUID),
		"operator_id":        strings.TrimSpace(input.OperatorID),
		"name":               strings.TrimSpace(input.Name),
		"description":        strings.TrimSpace(input.Description),
		"driver":             strings.TrimSpace(input.Driver),
		"folder":             strings.TrimSpace(input.Folder),
		"owner_subject_type": strings.TrimSpace(input.OwnerSubjectType),
		"owner_subject_id":   strings.TrimSpace(input.OwnerSubjectID),
		"tags":               append([]string(nil), input.Tags...),
		"upload_channel":     firstNonEmpty(string(input.UploadChannel), string(UploadChannelPresigned)),
		"external_url":       strings.TrimSpace(input.ExternalURL),
	}
	result, err := c.invoke(ctx, CapabilityMediaAssetsManage, "CreateMediaAsset", payload, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodeAsset(result.Data)
}

func (c *Client) GetAsset(ctx context.Context, input GetAssetInput) (*Asset, error) {
	result, err := c.invoke(ctx, CapabilityMediaAssetsRead, "GetMediaAsset", map[string]any{
		"tenant_uuid": strings.TrimSpace(input.TenantUUID),
		"uuid":        strings.TrimSpace(input.UUID),
	}, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodeAsset(result.Data)
}

func (c *Client) UpdateAsset(ctx context.Context, input UpdateAssetInput) (*Asset, error) {
	payload := map[string]any{
		"tenant_uuid": strings.TrimSpace(input.TenantUUID),
		"uuid":        strings.TrimSpace(input.UUID),
		"operator_id": strings.TrimSpace(input.OperatorID),
		"tags":        append([]string(nil), input.Tags...),
	}
	if input.Name != nil {
		payload["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		payload["description"] = strings.TrimSpace(*input.Description)
	}
	if input.BusinessStatus != nil {
		payload["business_status"] = string(*input.BusinessStatus)
	}
	result, err := c.invoke(ctx, CapabilityMediaAssetsRead, "UpdateMediaAsset", payload, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodeAsset(result.Data)
}

func (c *Client) PresignAsset(ctx context.Context, input PresignAssetInput) (*PresignAssetOutput, error) {
	payload := map[string]any{
		"tenant_uuid":        strings.TrimSpace(input.TenantUUID),
		"uuid":               strings.TrimSpace(input.UUID),
		"operator_id":        strings.TrimSpace(input.OperatorID),
		"action":             firstNonEmpty(string(input.Action), string(PresignActionDownload)),
		"expires_in_seconds": input.ExpiresInSeconds,
		"method":             strings.TrimSpace(input.Method),
		"metadata":           copyStringMap(input.Headers),
	}
	result, err := c.invoke(ctx, CapabilityMediaAssetsManage, "PresignMediaAsset", payload, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodePresign(result.Data)
}

func (c *Client) CreateAssetVariant(ctx context.Context, input CreateAssetVariantInput) (*AssetVariant, error) {
	payload := map[string]any{
		"tenant_uuid": strings.TrimSpace(input.TenantUUID),
		"uuid":        strings.TrimSpace(input.UUID),
		"variant":     strings.TrimSpace(input.Variant),
		"name":        strings.TrimSpace(input.Name),
		"driver":      strings.TrimSpace(input.Driver),
		"object_key":  strings.TrimSpace(input.ObjectKey),
		"size_bytes":  input.SizeBytes,
		"mime_type":   strings.TrimSpace(input.MimeType),
		"metadata":    copyStringMap(input.Metadata),
	}
	result, err := c.invoke(ctx, CapabilityMediaAssetsManage, "CreateMediaAssetVariant", payload, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodeAssetVariant(result.Data)
}

func (c *Client) PresignAssetVariant(ctx context.Context, input PresignAssetVariantInput) (*PresignAssetOutput, error) {
	payload := map[string]any{
		"tenant_uuid":        strings.TrimSpace(input.TenantUUID),
		"uuid":               strings.TrimSpace(input.UUID),
		"variant":            strings.TrimSpace(input.Variant),
		"operator_id":        strings.TrimSpace(input.OperatorID),
		"action":             firstNonEmpty(string(input.Action), string(PresignActionDownload)),
		"expires_in_seconds": input.ExpiresInSeconds,
		"method":             strings.TrimSpace(input.Method),
		"metadata":           copyStringMap(input.Headers),
	}
	result, err := c.invoke(ctx, CapabilityMediaAssetsManage, "PresignMediaAssetVariant", payload, input.TenantUUID, input.RequestID)
	if err != nil {
		return nil, err
	}
	return decodePresign(result.Data)
}

func (c *Client) UploadBytes(ctx context.Context, ticket *PresignAssetOutput, body io.Reader, contentType string) error {
	if c == nil || c.http == nil {
		return errors.New("media client unavailable")
	}
	if ticket == nil || strings.TrimSpace(ticket.URL) == "" {
		return errors.New("media upload ticket url is required")
	}
	method := strings.TrimSpace(ticket.Method)
	if method == "" {
		method = http.MethodPut
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSpace(ticket.URL), body)
	if err != nil {
		return err
	}
	for k, v := range ticket.Headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	if strings.TrimSpace(contentType) != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", strings.TrimSpace(contentType))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("media upload failed: status=%d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DownloadBytes(ctx context.Context, ticket *PresignAssetOutput) ([]byte, error) {
	if c == nil || c.http == nil {
		return nil, errors.New("media client unavailable")
	}
	if ticket == nil || strings.TrimSpace(ticket.URL) == "" {
		return nil, errors.New("media download ticket url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(ticket.URL), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range ticket.Headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("media download failed: status=%d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) invoke(ctx context.Context, capabilityID, action string, payload map[string]any, tenantUUID, requestID string) (*gateway.Response, error) {
	if c == nil || c.gateway == nil {
		return nil, errors.New("media gateway is not configured")
	}
	return c.gateway.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      capabilityID,
		Action:            action,
		PreferredProtocol: "grpc",
		Payload:           payload,
		RequestID:         strings.TrimSpace(requestID),
		TenantUUID:        strings.TrimSpace(tenantUUID),
	})
}

func decodeAsset(data map[string]any) (*Asset, error) {
	raw := unwrapData(data)
	if raw == nil {
		return nil, errors.New("media asset response data is empty")
	}
	var out Asset
	if err := decodeMap(raw, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.UUID) == "" {
		return nil, errors.New("media asset response missing uuid")
	}
	return &out, nil
}

func decodeAssetVariant(data map[string]any) (*AssetVariant, error) {
	raw := unwrapData(data)
	if raw == nil {
		return nil, errors.New("media asset variant response data is empty")
	}
	var out AssetVariant
	if err := decodeMap(raw, &out); err != nil {
		return nil, err
	}
	if rawMap, ok := raw.(map[string]any); ok {
		if strings.TrimSpace(out.AssetUUID) == "" {
			out.AssetUUID = firstStringFromMap(rawMap, "assetUuid", "asset_uuid", "AssetUUID")
		}
		if strings.TrimSpace(out.ObjectKey) == "" {
			out.ObjectKey = firstStringFromMap(rawMap, "objectKey", "object_key", "StorageKey")
		}
		if strings.TrimSpace(out.DownloadURL) == "" {
			out.DownloadURL = firstStringFromMap(rawMap, "downloadUrl", "download_url")
		}
	}
	if strings.TrimSpace(out.AssetUUID) == "" {
		return nil, errors.New("media asset variant response missing asset uuid")
	}
	if strings.TrimSpace(out.ObjectKey) == "" {
		return nil, errors.New("media asset variant response missing object key")
	}
	if strings.TrimSpace(out.Variant) == "" {
		return nil, errors.New("media asset variant response missing variant")
	}
	return &out, nil
}

func decodePresign(data map[string]any) (*PresignAssetOutput, error) {
	raw := unwrapData(data)
	if raw == nil {
		return nil, errors.New("media presign response data is empty")
	}
	var out PresignAssetOutput
	if err := decodeMap(raw, &out); err != nil {
		return nil, err
	}
	if rawMap, ok := raw.(map[string]any); ok {
		if out.ExpiresInSeconds == 0 {
			out.ExpiresInSeconds = uint32(firstNumberFromMap(rawMap, "expiresInSeconds", "expires_in_seconds"))
		}
		if strings.TrimSpace(out.ObjectKey) == "" {
			out.ObjectKey = firstStringFromMap(rawMap, "objectKey", "object_key", "StorageKey")
		}
	}
	if strings.TrimSpace(out.URL) == "" {
		return nil, errors.New("media presign response missing url")
	}
	return &out, nil
}

func unwrapData(data map[string]any) any {
	if data == nil {
		return nil
	}
	if nested, ok := data["data"]; ok {
		return nested
	}
	if payload, ok := data["payload"]; ok {
		if payloadMap, ok := payload.(map[string]any); ok {
			if nested, ok := payloadMap["data"]; ok {
				return nested
			}
		}
		return payload
	}
	if result, ok := data["result"]; ok {
		if resultMap, ok := result.(map[string]any); ok {
			if nested, ok := resultMap["data"]; ok {
				return nested
			}
		}
		return result
	}
	return data
}

func decodeMap(value any, out any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(out)
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.TrimSpace(k) != "" {
			out[k] = v
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstStringFromMap(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func firstNumberFromMap(data map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case json.Number:
			n, err := typed.Int64()
			if err == nil {
				return n
			}
		case int:
			return int64(typed)
		case int64:
			return typed
		case float64:
			return int64(typed)
		case string:
			n, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			if err == nil {
				return n
			}
		}
	}
	return 0
}

func DefaultUploadTTL() uint32 {
	return uint32((15 * time.Minute) / time.Second)
}
