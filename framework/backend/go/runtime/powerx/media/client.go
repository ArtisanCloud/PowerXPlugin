// Package media provides tenant-credential-scoped access to PowerX media
// capabilities. Unlike the legacy framework/media client, its public DTOs do
// not accept tenant_uuid: Core derives tenancy from the delegated credential.
package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

const CapabilityAssetsRead = "com.corex.media.assets.read"

// AssetReader is the business boundary used by plugin services and Host
// Contract Lab. Consumers must not use the generic capability Registry
// directly for media catalog reads.
type AssetReader interface {
	ListAssets(context.Context, ListAssetsInput) (*ListAssetsOutput, error)
}

type Client struct {
	registry powerxcapability.Registry
}

func NewClient(registry powerxcapability.Registry) *Client {
	return &Client{registry: registry}
}

type ListAssetsInput struct {
	Page     int
	PageSize int
	Keyword  string
}

type Asset struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

type ListAssetsOutput struct {
	Items    []Asset `json:"items"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	TraceID  string  `json:"trace_id,omitempty"`
}

func (c *Client) ListAssets(ctx context.Context, input ListAssetsInput) (*ListAssetsOutput, error) {
	if c == nil || c.registry == nil {
		return nil, fmt.Errorf("powerx media client is not configured")
	}
	payload := map[string]any{}
	if input.Page > 0 {
		payload["page"] = input.Page
	}
	if input.PageSize > 0 {
		payload["page_size"] = input.PageSize
	}
	if keyword := strings.TrimSpace(input.Keyword); keyword != "" {
		payload["keyword"] = keyword
	}
	invocation, err := c.registry.Invoke(ctx, powerxcapability.InvokeInput{
		CapabilityID: CapabilityAssetsRead,
		Payload:      payload,
	})
	if err != nil {
		return nil, err
	}
	if invocation == nil {
		return nil, fmt.Errorf("powerx media response is empty")
	}
	data := invocation.Result
	if len(data) == 0 {
		data = invocation.Payload
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode powerx media response: %w", err)
	}
	var output ListAssetsOutput
	if err := json.Unmarshal(encoded, &output); err != nil {
		return nil, fmt.Errorf("decode powerx media response: %w", err)
	}
	if output.Items == nil {
		output.Items = []Asset{}
	}
	output.TraceID = strings.TrimSpace(invocation.TraceID)
	return &output, nil
}
