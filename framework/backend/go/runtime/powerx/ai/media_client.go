package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

// DecodeHTTPError retains the published REST/Gateway diagnostic fields. Body is
// diagnostic evidence only and must not be rendered as a user-facing message.
func DecodeHTTPError(status int, headers http.Header, raw []byte) *HTTPError {
	var body struct {
		ErrorCode string          `json:"error_code"`
		TraceID   string          `json:"trace_id"`
		RequestID string          `json:"request_id"`
		Details   map[string]any  `json:"details"`
		Error     json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(raw, &body)
	var nested struct {
		ErrorCode string         `json:"error_code"`
		TraceID   string         `json:"trace_id"`
		RequestID string         `json:"request_id"`
		Details   map[string]any `json:"details"`
	}
	_ = json.Unmarshal(body.Error, &nested)
	if body.Details == nil {
		body.Details = nested.Details
	}
	first := func(values ...string) string {
		for _, v := range values {
			if v != "" {
				return v
			}
		}
		return ""
	}
	detailString := func(key string) string { value, _ := body.Details[key].(string); return value }
	reason := hostcontract.ParseReasonCode(raw, "AI_UPSTREAM_DEPENDENCY")
	upstream, _ := body.Details["upstream_status"].(float64)
	return &HTTPError{StatusCode: status, ErrorCode: first(body.ErrorCode, nested.ErrorCode, reason), ReasonCode: reason, Details: body.Details,
		TraceID: first(headers.Get("X-Trace-ID"), body.TraceID, nested.TraceID, detailString("trace_id")), RequestID: first(headers.Get("X-Request-ID"), body.RequestID, nested.RequestID),
		UpstreamStatus: int(upstream), UpstreamCode: detailString("upstream_code"), ProviderTraceID: detailString("provider_trace_id"), Body: string(raw)}
}

// invokeMediaBinding sends only a typed body. The Core registration owns the
// endpoint/method and tenant environment; the caller cannot override any of them.
func (c *Client) invokeMediaBinding(ctx context.Context, capability string, body any, out any) error {
	request := struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           struct {
			Body any `json:"body"`
		} `json:"payload"`
	}{CapabilityID: capability, PreferredProtocol: "core_internal"}
	request.Payload.Body = body
	var data struct {
		ProtocolUsed string `json:"protocol_used"`
		FallbackUsed bool   `json:"fallback_used"`
		Payload      struct {
			Result json.RawMessage `json:"result"`
		} `json:"payload"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/tenant/invocations", request, &data); err != nil {
		return err
	}
	if data.FallbackUsed || data.ProtocolUsed != "core_internal" || len(data.Payload.Result) == 0 || string(data.Payload.Result) == "null" {
		return errors.New("AI_BINDING_RESPONSE_INVALID")
	}
	return json.Unmarshal(data.Payload.Result, out)
}

func ValidateMediaCatalogFilter(filter MediaCatalogFilter) error {
	switch filter.Capability {
	case "", VisionUnderstanding, ImageGeneration, VideoGeneration, SpeechSynthesis:
		return nil
	default:
		return errors.New("AI_MODEL_CAPABILITY_UNSUPPORTED")
	}
}

func (c *Client) ListMediaModels(ctx context.Context, filter MediaCatalogFilter) (*MediaModelCatalog, error) {
	if err := ValidateMediaCatalogFilter(filter); err != nil {
		return nil, err
	}
	var out MediaModelCatalog
	body := struct {
		Operation string `json:"operation"`
		MediaCatalogFilter
	}{"list_models", filter}
	if err := c.invokeMediaBinding(ctx, CapabilityMediaModels, body, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListMediaProviders(ctx context.Context, capability string) (*MediaProviderCatalog, error) {
	if err := ValidateMediaCatalogFilter(MediaCatalogFilter{Capability: capability}); err != nil {
		return nil, err
	}
	var out MediaProviderCatalog
	body := struct {
		Operation  string `json:"operation"`
		Capability string `json:"capability,omitempty"`
	}{"list_providers", capability}
	if err := c.invokeMediaBinding(ctx, CapabilityMediaProviders, body, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) mediaInvoke(ctx context.Context, modality string, input MediaInvokeInput, out any) error {
	if strings.TrimSpace(input.ModelKey) == "" || len(input.Inputs) == 0 {
		return errors.New("AI_REQUEST_INVALID")
	}
	if modality != "vlm" && !idempotencyPattern.MatchString(input.IdempotencyKey) {
		return errors.New("AI_IDEMPOTENCY_KEY_REQUIRED")
	}
	body := struct {
		Operation       string `json:"operation"`
		ContractVersion int    `json:"contract_version"`
		MediaInvokeInput
	}{"invoke", 2, input}
	return c.invokeMediaBinding(ctx, "com.corex.ai."+modality+".invoke", body, out)
}

func (c *Client) VLMInvoke(ctx context.Context, in MediaInvokeInput) (*AnalysisResult, error) {
	var out AnalysisResult
	if err := c.mediaInvoke(ctx, "vlm", in, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != 2 || out.Type != "analysis" || out.Provider == "" || out.Model == "" || out.TraceID == "" {
		return nil, errors.New("AI_ANALYSIS_RESPONSE_INVALID")
	}
	return &out, nil
}
func (c *Client) generateMedia(ctx context.Context, modality, capability string, in MediaInvokeInput) (*MediaTask, error) {
	var out MediaTask
	if err := c.mediaInvoke(ctx, modality, in, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	if out.ModelKey != in.ModelKey || out.Capability != capability {
		return nil, errors.New("AI_TASK_RESPONSE_INVALID")
	}
	return &out, nil
}
func (c *Client) ImageInvoke(ctx context.Context, in MediaInvokeInput) (*MediaTask, error) {
	return c.generateMedia(ctx, "image", ImageGeneration, in)
}
func (c *Client) VideoInvoke(ctx context.Context, in MediaInvokeInput) (*MediaTask, error) {
	return c.generateMedia(ctx, "video", VideoGeneration, in)
}
func (c *Client) TTSInvoke(ctx context.Context, in MediaInvokeInput) (*MediaTask, error) {
	return c.generateMedia(ctx, "tts", SpeechSynthesis, in)
}

func (c *Client) mediaTaskOperation(ctx context.Context, capability, operation, taskUUID string) (*MediaTask, error) {
	if !ValidMediaTaskUUID(taskUUID) {
		return nil, errors.New("AI_TASK_UUID_INVALID")
	}
	body := struct {
		Operation string `json:"operation"`
		TaskUUID  string `json:"task_uuid"`
	}{operation, taskUUID}
	var out MediaTask
	if err := c.invokeMediaBinding(ctx, capability, body, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	if out.TaskUUID != taskUUID {
		return nil, errors.New("AI_TASK_RESPONSE_INVALID")
	}
	return &out, nil
}
func (c *Client) GetMediaTask(ctx context.Context, taskUUID string) (*MediaTask, error) {
	return c.mediaTaskOperation(ctx, CapabilityMediaTasksRead, "get", taskUUID)
}
func (c *Client) CancelMediaTask(ctx context.Context, taskUUID string) (*MediaTask, error) {
	return c.mediaTaskOperation(ctx, CapabilityMediaTasksCancel, "cancel", taskUUID)
}
