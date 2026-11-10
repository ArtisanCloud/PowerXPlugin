package aisettings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
)

const CapabilityAIConnectionTest = "com.corex.ai.connection.test"

// ConnectionTestInput uses Core's saved connection and native optional model.
// Environment, credentials, endpoint and generation inputs are never accepted.
type ConnectionTestInput struct {
	Modality string `json:"modality"`
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
}
type ConnectionVerification struct {
	Provider            string `json:"provider"`
	Probe               string `json:"probe"`
	EndpointReachable   bool   `json:"endpoint_reachable"`
	APIKeyVerified      bool   `json:"api_key_verified"`
	Authentication      string `json:"authentication"`
	ModelAccessVerified bool   `json:"model_access_verified"`
	GenerationSubmitted bool   `json:"generation_submitted"`
}
type ConnectionTestResult struct {
	ContractVersion int                    `json:"contract_version"`
	OK              bool                   `json:"ok"`
	Environment     string                 `json:"env"`
	Modality        string                 `json:"modality"`
	Verification    ConnectionVerification `json:"verification"`
}
type ConnectionTester interface {
	TestConnection(context.Context, ConnectionTestInput, string) (*ConnectionTestResult, error)
}

func (in ConnectionTestInput) Validate() error {
	if strings.TrimSpace(in.Provider) == "" || in.Provider != strings.TrimSpace(in.Provider) {
		return errors.New("AI_REQUEST_INVALID")
	}
	switch in.Modality {
	case "llm", "vlm", "image", "video", "embedding", "audio_tts", "audio_asr", "model3d", "rerank":
		return nil
	default:
		return errors.New("AI_REQUEST_INVALID")
	}
}

func connectionContractError() error {
	return &powerxai.HTTPError{StatusCode: 502, ErrorCode: "AI_CONNECTION_CONTRACT_INVALID", ReasonCode: "AI_CONNECTION_CONTRACT_INVALID", Details: map[string]any{"generation_submitted": false}}
}

func (out ConnectionTestResult) Validate(in ConnectionTestInput) error {
	v := out.Verification
	if out.ContractVersion != 1 || !out.OK || strings.TrimSpace(out.Environment) == "" || out.Modality != in.Modality || v.Provider != in.Provider || !v.EndpointReachable || v.ModelAccessVerified || v.GenerationSubmitted {
		return connectionContractError()
	}
	switch v.Authentication {
	case "api_key":
		if !v.APIKeyVerified || (v.Probe != "models" && v.Probe != "key_info") {
			return connectionContractError()
		}
	case "not_required":
		if v.APIKeyVerified || v.Probe != "ollama_tags" {
			return connectionContractError()
		}
	default:
		return connectionContractError()
	}
	return nil
}

// TestConnection is an explicit diagnostic operation. It does not select or
// alter the business runtime, retry, approve grants or call an Admin endpoint.
func (c *Client) TestConnection(ctx context.Context, input ConnectionTestInput, requestID string) (*ConnectionTestResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if c == nil || c.invoker == nil {
		return nil, errors.New("AI_CLIENT_UNAVAILABLE")
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	body := struct {
		ContractVersion int    `json:"contract_version"`
		Operation       string `json:"operation"`
		ConnectionTestInput
	}{1, "test_connection", input}
	response, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: CapabilityAIConnectionTest, PreferredProtocol: "core_internal", TenantUUID: c.tenantUUID, RequestID: requestID, Payload: map[string]any{"body": body}})
	if err != nil {
		var invocation *gateway.InvocationError
		if errors.As(err, &invocation) {
			headers := http.Header{}
			headers.Set("X-Trace-ID", invocation.TraceID)
			headers.Set("X-Request-ID", invocation.UpstreamRequestID)
			return nil, powerxai.DecodeHTTPError(invocation.StatusCode, headers, invocation.Body)
		}
		return nil, err
	}
	if response == nil || response.Data == nil || response.Data["protocol_used"] != "core_internal" || response.Data["status"] != "completed" || response.Data["fallback_used"] != false {
		return nil, connectionContractError()
	}
	raw, err := json.Marshal(response.Data["payload"])
	if err != nil {
		return nil, connectionContractError()
	}
	var payload struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return nil, connectionContractError()
	}
	var out ConnectionTestResult
	decoder := json.NewDecoder(bytes.NewReader(payload.Result))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, connectionContractError()
	}
	var fields map[string]json.RawMessage
	var verification map[string]json.RawMessage
	if json.Unmarshal(payload.Result, &fields) != nil || json.Unmarshal(fields["verification"], &verification) != nil {
		return nil, connectionContractError()
	}
	for _, name := range []string{"endpoint_reachable", "api_key_verified", "model_access_verified", "generation_submitted"} {
		var value *bool
		if json.Unmarshal(verification[name], &value) != nil || value == nil {
			return nil, connectionContractError()
		}
	}
	if err := out.Validate(input); err != nil {
		return nil, err
	}
	return &out, nil
}
