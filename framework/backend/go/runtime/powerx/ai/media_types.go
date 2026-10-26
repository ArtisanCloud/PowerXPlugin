package ai

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

const MediaContractVersion = 2
const (
	VisionUnderstanding        = "vision_understanding"
	ImageGeneration            = "image_generation"
	VideoGeneration            = "video_generation"
	SpeechSynthesis            = "speech_synthesis"
	CapabilityMediaModels      = "com.corex.ai.catalog.models.read"
	CapabilityMediaProviders   = "com.corex.ai.catalog.providers.read"
	CapabilityMediaTasksRead   = "com.corex.ai.media.tasks.read"
	CapabilityMediaTasksCancel = "com.corex.ai.media.tasks.cancel"
)

// MediaInvokeInput carries a business-owned attempt key. Clients never generate
// or replace that key, including after a failed submission or stream reconnect.
type MediaInvokeInput struct {
	ModelKey       string         `json:"model_key"`
	Inputs         []ContentItem  `json:"inputs"`
	Params         map[string]any `json:"params,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
}

type MediaCatalogFilter struct {
	Provider   string `json:"provider,omitempty"`
	App        string `json:"app,omitempty"`
	Capability string `json:"capability,omitempty"`
}

// ParameterSchema is the published JSON Schema subset, preserving zero/false
// defaults and zero limits. Core remains authoritative for parameter validation.
type ParameterSchema struct {
	Type                 string                     `json:"type"`
	Properties           map[string]ParameterSchema `json:"properties,omitempty"`
	AdditionalProperties *bool                      `json:"additionalProperties,omitempty"`
	Enum                 []any                      `json:"enum,omitempty"`
	Minimum              *float64                   `json:"minimum,omitempty"`
	Maximum              *float64                   `json:"maximum,omitempty"`
	Default              any                        `json:"default,omitempty"`
}
type MediaModelOperation struct {
	Driver          string                     `json:"driver"`
	InputTypes      []string                   `json:"input_types"`
	MaxImages       int                        `json:"max_images"`
	MaxVideos       int                        `json:"max_videos"`
	Parameters      map[string]ParameterSchema `json:"parameters"`
	ParameterSchema ParameterSchema            `json:"parameter_schema"`
	Defaults        map[string]any             `json:"defaults"`
	Async           bool                       `json:"async"`
	Available       bool                       `json:"available"`
	ReasonCode      string                     `json:"reason_code,omitempty"`
}
type MediaModel struct {
	ModelKey     string                         `json:"model_key"`
	Provider     string                         `json:"provider"`
	Model        string                         `json:"model"`
	App          string                         `json:"app,omitempty"`
	Label        string                         `json:"label"`
	Capabilities []string                       `json:"capabilities"`
	Configured   bool                           `json:"configured"`
	Available    bool                           `json:"available"`
	ReasonCode   string                         `json:"reason_code,omitempty"`
	Operations   map[string]MediaModelOperation `json:"operations"`
}
type MediaModelCatalog struct {
	ContractVersion int          `json:"contract_version"`
	Items           []MediaModel `json:"items"`
}
type MediaProvider struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
}
type MediaProviderCatalog struct {
	ContractVersion int             `json:"contract_version"`
	Providers       []MediaProvider `json:"providers"`
}

func (catalog MediaModelCatalog) Validate() error {
	if catalog.ContractVersion != MediaContractVersion || catalog.Items == nil {
		return errors.New("AI_CATALOG_RESPONSE_INVALID")
	}
	for _, model := range catalog.Items {
		if model.ModelKey == "" || model.Provider == "" || model.Model == "" || model.Label == "" || len(model.Operations) == 0 || len(model.Capabilities) == 0 {
			return errors.New("AI_CATALOG_RESPONSE_INVALID")
		}
		for _, capability := range model.Capabilities {
			operation, ok := model.Operations[capability]
			if !ok || operation.InputTypes == nil || operation.Defaults == nil || operation.ParameterSchema.Type != "object" || operation.ParameterSchema.Properties == nil || operation.ParameterSchema.AdditionalProperties == nil || *operation.ParameterSchema.AdditionalProperties {
				return errors.New("AI_CATALOG_RESPONSE_INVALID")
			}
		}
	}
	return nil
}
func (catalog MediaProviderCatalog) Validate() error {
	if catalog.ContractVersion != MediaContractVersion || catalog.Providers == nil {
		return errors.New("AI_CATALOG_RESPONSE_INVALID")
	}
	for _, provider := range catalog.Providers {
		if provider.Provider == "" || provider.Label == "" {
			return errors.New("AI_CATALOG_RESPONSE_INVALID")
		}
	}
	return nil
}

type AnalysisResult struct {
	ContractVersion int            `json:"contract_version"`
	Type            string         `json:"type"`
	Text            string         `json:"text"`
	Provider        string         `json:"provider"`
	Model           string         `json:"model"`
	Usage           map[string]any `json:"usage,omitempty"`
	TraceID         string         `json:"trace_id"`
}
type MediaAssetResult struct {
	MediaAssetUUID    string `json:"media_asset_uuid"`
	Type              string `json:"type"`
	MimeType          string `json:"mime_type"`
	SizeBytes         int64  `json:"size_bytes"`
	Checksum          string `json:"checksum"`
	ChecksumAlgorithm string `json:"checksum_algorithm"`
}
type MediaTask struct {
	ContractVersion   int                `json:"contract_version"`
	TaskUUID          string             `json:"task_uuid"`
	Status            string             `json:"status"`
	Phase             string             `json:"phase"`
	ModelKey          string             `json:"model_key"`
	Capability        string             `json:"capability"`
	Revision          int64              `json:"revision"`
	TraceID           string             `json:"trace_id"`
	ErrorCode         string             `json:"error_code,omitempty"`
	ProviderErrorCode string             `json:"provider_error_code,omitempty"`
	UpstreamStatus    int                `json:"upstream_status,omitempty"`
	ProviderTraceID   string             `json:"provider_trace_id,omitempty"`
	CancelStatus      string             `json:"cancel_status"`
	Assets            []MediaAssetResult `json:"assets"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
}

func (t MediaTask) Terminal() bool {
	return t.Status == "succeeded" || t.Status == "failed" || t.Status == "cancelled"
}
func ValidMediaTaskUUID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed != uuid.Nil && parsed.String() == id
}
func (t MediaTask) Validate() error {
	if t.ContractVersion != 2 || !ValidMediaTaskUUID(t.TaskUUID) || t.Revision < 1 || t.ModelKey == "" || t.TraceID == "" || t.Assets == nil || t.CreatedAt.IsZero() || t.UpdatedAt.IsZero() {
		return errors.New("AI_TASK_RESPONSE_INVALID")
	}
	if t.Status != "queued" && t.Status != "running" && !t.Terminal() {
		return errors.New("AI_TASK_RESPONSE_INVALID")
	}
	switch t.Phase {
	case "queued", "submitting", "waiting_provider", "materializing", "completed":
	default:
		return errors.New("AI_TASK_RESPONSE_INVALID")
	}
	switch t.CancelStatus {
	case "none", "requested", "confirmed", "unsupported":
	default:
		return errors.New("AI_TASK_RESPONSE_INVALID")
	}
	switch t.Capability {
	case ImageGeneration, VideoGeneration, SpeechSynthesis:
	default:
		return errors.New("AI_TASK_RESPONSE_INVALID")
	}
	for _, asset := range t.Assets {
		if !ValidMediaTaskUUID(asset.MediaAssetUUID) || asset.SizeBytes < 1 || asset.MimeType == "" || asset.ChecksumAlgorithm != "sha256" || !checksumPattern.MatchString(asset.Checksum) {
			return errors.New("AI_ASSET_RESPONSE_INVALID")
		}
		if asset.Type != "image" && asset.Type != "video" && asset.Type != "audio" {
			return errors.New("AI_ASSET_RESPONSE_INVALID")
		}
	}
	if t.Status == "succeeded" && len(t.Assets) == 0 {
		return errors.New("AI_ASSET_RESPONSE_INVALID")
	}
	return nil
}
func (t MediaTask) EventID() string { return fmt.Sprintf("%s:%d", t.TaskUUID, t.Revision) }

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var checksumPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
