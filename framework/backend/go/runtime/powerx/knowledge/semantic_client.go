package knowledge

import (
	"context"
	"encoding/json"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"net/http"
	"net/url"
	"strings"
)

const semanticReadCapability = "com.corex.knowledge.retrieval.read"
const documentManageCapability = "com.corex.knowledge.document.manage"

// invokeContract is private and accepts only fixed typed operations built here.
// Registry result/payload is a documented envelope union, not a transport fallback.
func (c *Client) invokeContract(ctx context.Context, capability string, operation, output any) (responseMeta, error) {
	request := struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           struct {
			Method   string `json:"method"`
			Endpoint string `json:"endpoint"`
			Body     any    `json:"body"`
		} `json:"payload"`
	}{CapabilityID: capability, PreferredProtocol: "core_internal"}
	request.Payload.Method = "INVOKE"
	switch capability {
	case semanticReadCapability:
		request.Payload.Endpoint = "core://knowledge/retrieval"
	case documentManageCapability:
		request.Payload.Endpoint = "core://knowledge/documents"
	case fw.ContentOverviewCapability:
		request.Payload.Endpoint = "core://knowledge/content-overview"
	default:
		return responseMeta{}, fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	request.Payload.Body = operation
	var response struct {
		Result    json.RawMessage `json:"result"`
		Payload   json.RawMessage `json:"payload"`
		TraceID   string          `json:"trace_id"`
		RequestID string          `json:"request_id"`
	}
	meta, err := c.requestMeta(ctx, http.MethodPost, "/api/v1/tenant/invocations", request, &response)
	if err != nil {
		return meta, err
	}
	meta.InvocationTraceID = response.TraceID
	if meta.RequestID == "" {
		meta.RequestID = response.RequestID
	}
	business := response.Result
	if len(business) == 0 || string(business) == "null" {
		business = response.Payload
	}
	if len(business) == 0 || string(business) == "null" {
		return meta, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
	}
	if err := json.Unmarshal(business, output); err != nil {
		return meta, fw.WrapError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response", err)
	}
	return meta, nil
}
func validateSpace(space string) error {
	if !fw.SemanticUUID(space) {
		return fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	return nil
}
func (c *Client) GetSemanticCapabilities(ctx context.Context, space string) (*fw.SemanticCapabilities, error) {
	if err := validateSpace(space); err != nil {
		return nil, err
	}
	var result fw.SemanticCapabilities
	var err error
	if c.authScheme == "ApiKey" {
		err = c.request(ctx, http.MethodGet, "/api/v1/tenant/knowledge/spaces/"+url.PathEscape(space)+"/semantic-index", nil, &result)
	} else {
		var out struct {
			Capabilities *fw.SemanticCapabilities `json:"capabilities"`
		}
		_, err = c.invokeContract(ctx, semanticReadCapability, struct {
			Operation string `json:"operation"`
			SpaceUUID string `json:"space_uuid"`
		}{"capabilities", space}, &out)
		if err == nil && out.Capabilities == nil {
			return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
		}
		if out.Capabilities != nil {
			result = *out.Capabilities
		}
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	if result.Schema != fw.SemanticCapabilitiesSchema || result.Indexed && !result.Ready || result.Ready && !result.Configured || result.Configured && result.Generation == nil {
		return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
	}
	return &result, nil
}
func (c *Client) ConfigureSemanticIndex(ctx context.Context, space string, input fw.SemanticConfigureInput) (*fw.SemanticGeneration, error) {
	if err := validateSpace(space); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.EmbeddingProfileKey) == "" || input.ExpectedConfigurationGeneration != "" && !fw.SemanticUUID(input.ExpectedConfigurationGeneration) {
		return nil, fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	var result fw.SemanticGeneration
	var err error
	if c.authScheme == "ApiKey" {
		err = c.request(ctx, http.MethodPost, "/api/v1/tenant/knowledge/spaces/"+url.PathEscape(space)+"/semantic-index", input, &result)
	} else {
		var out struct {
			Item *fw.SemanticGeneration `json:"item"`
		}
		_, err = c.invokeContract(ctx, documentManageCapability, struct {
			Operation     string                    `json:"operation"`
			SpaceUUID     string                    `json:"space_uuid"`
			Configuration fw.SemanticConfigureInput `json:"configuration"`
		}{"configure_semantic_index", space, input}, &out)
		if err == nil && out.Item == nil {
			return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
		}
		if out.Item != nil {
			result = *out.Item
		}
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	if result.SpaceUUID != space || !fw.SemanticUUID(result.EmbeddingProfile.UUID) || result.EmbeddingProfile.Version < 1 || result.Dimensions < 1 || result.ModelRevision == "" || !fw.SemanticUUID(result.ConfigurationGeneration) {
		return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
	}
	return &result, nil
}
func (c *Client) QuerySemanticKnowledge(ctx context.Context, query fw.SemanticQuery) (*fw.SemanticResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	var result fw.SemanticResult
	var meta responseMeta
	var err error
	if c.authScheme == "ApiKey" {
		meta, err = c.requestMeta(ctx, http.MethodPost, "/api/v1/tenant/knowledge/retrieval/query", query, &result)
	} else {
		var out struct {
			Result *fw.SemanticResult `json:"result"`
		}
		meta, err = c.invokeContract(ctx, semanticReadCapability, struct {
			Operation string           `json:"operation"`
			Query     fw.SemanticQuery `json:"query"`
		}{"query", query}, &out)
		if err == nil && out.Result == nil {
			return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
		}
		if out.Result != nil {
			result = *out.Result
		}
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	if err := result.Validate(query); err != nil {
		return nil, err
	}
	result.InvocationTraceID = meta.InvocationTraceID
	result.RequestID = meta.RequestID
	if result.TraceID == "" {
		result.TraceID = meta.TraceID
	}
	return &result, nil
}
func (c *Client) SetDocumentVisibility(ctx context.Context, space, document string, input fw.SemanticVisibilityInput) (*fw.SemanticVisibility, error) {
	if validateSpace(space) != nil || !fw.SemanticUUID(document) || input.Queryable == nil || input.ExpectedEpoch != "" && !fw.SemanticUUID(input.ExpectedEpoch) {
		return nil, fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	var result fw.SemanticVisibility
	var err error
	if c.authScheme == "ApiKey" {
		err = c.request(ctx, http.MethodPatch, "/api/v1/tenant/knowledge/spaces/"+url.PathEscape(space)+"/documents/"+url.PathEscape(document)+"/visibility", input, &result)
	} else {
		_, err = c.invokeContract(ctx, documentManageCapability, struct {
			Operation    string                     `json:"operation"`
			SpaceUUID    string                     `json:"space_uuid"`
			DocumentUUID string                     `json:"document_uuid"`
			Visibility   fw.SemanticVisibilityInput `json:"visibility"`
		}{"set_document_visibility", space, document, input}, &result)
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	if result.DocumentUUID != document || result.Queryable != *input.Queryable || !fw.SemanticUUID(result.VisibilityEpoch) {
		return nil, fw.NewError(fw.CodeInvalidResponse, "knowledge.semantic.invalid_response")
	}
	return &result, nil
}

type hostDocumentInput struct {
	Title          string                        `json:"title"`
	URI            string                        `json:"uri"`
	Content        string                        `json:"content"`
	ContentType    string                        `json:"content_type"`
	Checksum       string                        `json:"checksum"`
	Version        string                        `json:"version"`
	Tags           []string                      `json:"tags"`
	Ingestion      *fw.DocumentIngestionSettings `json:"ingestion,omitempty"`
	Indexing       *fw.SemanticIndexingSettings  `json:"indexing,omitempty"`
	Artifacts      []fw.SemanticArtifact         `json:"artifacts,omitempty"`
	ExternalRef    *fw.SemanticExternalRef       `json:"external_ref,omitempty"`
	IdempotencyKey string                        `json:"idempotency_key,omitempty"`
}
