// Package knowledge provides the typed delegated client for Core QA bridge APIs.
package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
	"github.com/google/uuid"
)

type Config struct {
	BaseURL, BearerToken, STSClientID, STSClientSecret, STSTokenURL string
	Timeout                                                         time.Duration
}
type tokenProvider interface {
	Token(context.Context) (string, error)
}
type TokenProvider = tokenProvider

type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type Client struct {
	baseURL    string
	http       *http.Client
	token      tokenProvider
	authScheme string
}

// NewClientWithAPIKey uses a server-configured credential for Local + proxy
// diagnostics. It never forwards the plugin's local user token to Core.
func NewClientWithAPIKey(cfg Config, apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("knowledge.api_key_required")
	}
	c, err := NewClientWithTokenProvider(cfg, staticToken(strings.TrimSpace(apiKey)), httpClient)
	if err != nil {
		return nil, err
	}
	c.authScheme = "ApiKey"
	return c, nil
}

// QABridge is the Framework boundary for the currently published knowledge
// retrieval-plan and memory-snapshot operations.
type QABridge interface {
	RetrievalPlan(context.Context, RetrievalPlanInput) (*RetrievalPlan, error)
	UpsertMemorySnapshot(context.Context, MemorySnapshotInput) (*MemorySnapshot, error)
}

type RetrievalPlanInput struct {
	Intent          string   `json:"intent"`
	DomainTags      []string `json:"domainTags,omitempty"`
	SessionID       string   `json:"sessionId,omitempty"`
	LatencyBudgetMs int      `json:"latencyBudgetMs,omitempty"`
}
type PlanSpace struct {
	SpaceUUID        string  `json:"spaceId"`
	SpaceName        string  `json:"spaceName"`
	Strategy         string  `json:"strategy"`
	CitationCoverage float64 `json:"citationCoverage"`
	DegradeReason    string  `json:"degradeReason,omitempty"`
}
type RetrievalPlan struct {
	TenantUUID      string      `json:"tenant_uuid"`
	Intent          string      `json:"intent"`
	CandidateSpaces []PlanSpace `json:"candidateSpaces"`
	SessionID       string      `json:"sessionId"`
	TraceID         string      `json:"-"`
}
type MemoryCitation struct {
	ChunkID     string   `json:"chunkId"`
	SpaceUUID   string   `json:"spaceId"`
	Status      string   `json:"status"`
	Citations   []string `json:"citations"`
	SourceType  string   `json:"sourceType"`
	Confidence  float64  `json:"confidence"`
	DeltaReason string   `json:"deltaReason"`
}
type MemorySnapshotInput struct {
	SessionID string           `json:"sessionId"`
	Updates   []MemoryCitation `json:"updates,omitempty"`
	TraceID   string           `json:"traceId,omitempty"`
}
type MemorySnapshot struct {
	TenantUUID string           `json:"tenant_uuid"`
	SessionID  string           `json:"sessionId"`
	Citations  []MemoryCitation `json:"citations"`
	TraceID    string           `json:"traceId,omitempty"`
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx knowledge base_url is required")
	}
	c := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")}
	if httpClient != nil {
		c.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		c.http = &http.Client{Timeout: timeout}
	}
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		c.token = staticToken(token)
		return c, nil
	}
	client, err := sts.NewClient(sts.Config{TokenEndpoint: cfg.STSTokenURL, ClientID: cfg.STSClientID, ClientSecret: cfg.STSClientSecret})
	if err != nil {
		return nil, fmt.Errorf("powerx knowledge delegated sts configuration: %w", err)
	}
	c.token = stsToken{client}
	return c, nil
}

// NewClientWithTokenProvider shares Skeleton's delegated STS flow instead of
// creating a second token configuration for the QA bridge.
func NewClientWithTokenProvider(cfg Config, provider TokenProvider, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("powerx knowledge base_url is required")
	}
	if provider == nil {
		return nil, errors.New("powerx knowledge token provider is required")
	}
	c := &Client{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), token: provider}
	if httpClient != nil {
		c.http = httpClient
	} else {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		c.http = &http.Client{Timeout: timeout}
	}
	return c, nil
}
func (c *Client) RetrievalPlan(ctx context.Context, input RetrievalPlanInput) (*RetrievalPlan, error) {
	var out RetrievalPlan
	if err := c.do(ctx, "/api/v1/openapi/knowledge-spaces/qa/retrieval-plan", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) UpsertMemorySnapshot(ctx context.Context, input MemorySnapshotInput) (*MemorySnapshot, error) {
	var out MemorySnapshot
	if err := c.do(ctx, "/api/v1/openapi/knowledge-spaces/qa/memory-snapshot", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) do(ctx context.Context, path string, input, output any) error {
	return c.request(ctx, http.MethodPost, path, input, output)
}

type responseMeta struct{ TraceID, RequestID, InvocationTraceID string }

func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	_, err := c.requestMeta(ctx, method, path, input, output)
	return err
}
func (c *Client) requestMeta(ctx context.Context, method, path string, input, output any) (responseMeta, error) {
	meta := responseMeta{}
	if c == nil || c.http == nil || c.token == nil {
		return meta, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge.host.unavailable")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return meta, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return meta, err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	token, err := c.token.Token(ctx)
	if err != nil {
		return meta, err
	}
	req.Header.Set("X-Trace-ID", uuid.NewString())
	req.Header.Set("X-Request-ID", uuid.NewString())
	scheme := c.authScheme
	if scheme == "" {
		scheme = "Bearer"
	}
	req.Header.Set("Authorization", scheme+" "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return meta, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return meta, err
	}
	var envelope struct {
		Data      json.RawMessage `json:"data"`
		TraceID   string          `json:"trace_id"`
		RequestID string          `json:"request_id"`
		Details   map[string]any  `json:"details"`
	}
	_ = json.Unmarshal(raw, &envelope)
	meta.TraceID = resp.Header.Get("X-Trace-Id")
	if meta.TraceID == "" {
		meta.TraceID = envelope.TraceID
	}
	if meta.TraceID == "" {
		meta.TraceID = req.Header.Get("X-Trace-ID")
	}
	meta.RequestID = envelope.RequestID
	if meta.RequestID == "" {
		meta.RequestID = resp.Header.Get("X-Request-Id")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return meta, &HTTPError{StatusCode: resp.StatusCode, ReasonCode: hostcontract.ParseReasonCode(raw, "KNOWLEDGE_UPSTREAM_DEPENDENCY"), TraceID: meta.TraceID, RequestID: meta.RequestID, Details: fwknowledge.RedactMap(envelope.Details), Body: string(raw)}
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return meta, fwknowledge.NewError(fwknowledge.CodeInvalidResponse, "knowledge.semantic.invalid_response")
	}
	if err := json.Unmarshal(envelope.Data, output); err != nil {
		return meta, fwknowledge.WrapError(fwknowledge.CodeInvalidResponse, "knowledge.semantic.invalid_response", err)
	}
	return meta, nil
}

type HTTPError struct {
	RequestID  string
	Details    map[string]any
	TraceID    string
	StatusCode int
	ReasonCode string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("powerx knowledge request failed: status=%d", e.StatusCode)
}

// DelegatedCapabilities declares exactly the published tenant Knowledge Host
// Contract. Unsupported filters and document-level rebuilds fail explicitly.
func (c *Client) DelegatedCapabilities(context.Context) fwknowledge.ProviderCapabilities {
	return fwknowledge.BasicCapabilities("powerx_knowledge_host", fwknowledge.ProviderModeDelegated,
		fwknowledge.OperationCatalog, fwknowledge.OperationCreate,
		fwknowledge.OperationRetrieve, fwknowledge.OperationSearch, fwknowledge.OperationUpsert,
		fwknowledge.OperationDelete, fwknowledge.OperationReindex, fwknowledge.OperationHealth,
		fwknowledge.OperationSemanticCapabilities, fwknowledge.OperationSemanticQuery, fwknowledge.OperationSemanticConfigure, fwknowledge.OperationDocumentVisibility)
}

func (c *Client) ListKnowledgeSpaces(ctx context.Context, input fwknowledge.ListSpacesInput) ([]fwknowledge.KnowledgeSpace, error) {
	input = input.Normalized()
	if input.Status != "" || input.Visibility != "" || input.PluginID != "" || len(input.Filters) != 0 {
		return nil, fwknowledge.Unsupported(fwknowledge.OperationRetrieve)
	}
	var response struct {
		Items []struct {
			SpaceUUID string `json:"space_uuid"`
			Name      string `json:"name"`
			Status    string `json:"status"`
		} `json:"items"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/tenant/knowledge/spaces", nil, &response); err != nil {
		return nil, mapHostError(err)
	}
	spaces := make([]fwknowledge.KnowledgeSpace, 0, len(response.Items))
	for _, item := range response.Items {
		if strings.TrimSpace(item.SpaceUUID) == "" || strings.TrimSpace(item.Name) == "" {
			return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge host returned an invalid space")
		}
		spaces = append(spaces, fwknowledge.KnowledgeSpace{SpaceID: strings.TrimSpace(item.SpaceUUID), SpaceName: strings.TrimSpace(item.Name), TenantUUID: input.TenantUUID, Visibility: fwknowledge.VisibilityTenant, Status: strings.TrimSpace(item.Status)})
	}
	return spaces, nil
}

func (c *Client) SearchKnowledge(ctx context.Context, query fwknowledge.KnowledgeQuery) (*fwknowledge.KnowledgeSearchResult, error) {
	query = query.Normalized()
	if err := query.Validate(fwknowledge.TenantRequiredForVisibility(query.Visibility)); err != nil {
		return nil, err
	}
	if len(query.Tags) != 0 || len(query.Filters) != 0 || query.MinScore != 0 {
		return nil, fwknowledge.Unsupported(fwknowledge.OperationSearch)
	}
	var response struct {
		Items []struct {
			SpaceUUID    string   `json:"space_uuid"`
			DocumentUUID string   `json:"document_uuid"`
			Title        string   `json:"title"`
			URI          string   `json:"uri"`
			Excerpt      string   `json:"excerpt"`
			Tags         []string `json:"tags"`
		} `json:"items"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/v1/tenant/knowledge/search", map[string]any{"query": query.Query, "space_uuids": query.SpaceIDs, "limit": query.Limit}, &response); err != nil {
		return nil, mapHostError(err)
	}
	result := &fwknowledge.KnowledgeSearchResult{Provider: "powerx_knowledge_host", Chunks: make([]fwknowledge.KnowledgeChunk, 0, len(response.Items)), TraceID: query.TraceID}
	for _, item := range response.Items {
		if strings.TrimSpace(item.SpaceUUID) == "" || strings.TrimSpace(item.DocumentUUID) == "" || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Excerpt) == "" {
			return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge host returned an invalid citation")
		}
		citation := &fwknowledge.KnowledgeCitation{DocumentID: strings.TrimSpace(item.DocumentUUID), Title: strings.TrimSpace(item.Title), URI: strings.TrimSpace(item.URI), Provider: "powerx_knowledge_host", RetrievedAt: time.Now().UTC()}
		result.Chunks = append(result.Chunks, fwknowledge.KnowledgeChunk{DocumentID: citation.DocumentID, SpaceID: strings.TrimSpace(item.SpaceUUID), Text: strings.TrimSpace(item.Excerpt), Metadata: map[string]any{"tags": append([]string(nil), item.Tags...)}, Citation: citation, TenantUUID: query.TenantUUID})
	}
	return result, nil
}

func (c *Client) UpsertKnowledgeDocument(ctx context.Context, document fwknowledge.KnowledgeDocument) (*fwknowledge.KnowledgeIndexJob, error) {
	if err := document.ValidateSemantic(); err != nil {
		return nil, err
	}
	// Local configuration is not the published Host snapshot DTO.
	if document.Ingestion != nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeUnsupportedCapability, "knowledgeLab.workspace.unsupported")
	}
	if err := document.HostIngestion.Validate(); err != nil {
		return nil, err
	}
	document, err := fwknowledge.ValidateHostDocument(document)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(document.Content) == "" || strings.TrimSpace(document.ContentType) == "" || strings.TrimSpace(document.Checksum) == "" || strings.TrimSpace(document.Version) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge Host document requires content, content_type, checksum, and version")
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/spaces/" + url.PathEscape(document.SpaceID) + "/documents"
	payload := hostDocumentInput{Title: document.Title, URI: document.URI, Content: document.Content, ContentType: document.ContentType, Checksum: document.Checksum, Version: document.Version, Tags: document.Tags, Ingestion: document.HostIngestion, Indexing: document.Indexing, Artifacts: document.Artifacts, ExternalRef: document.ExternalRef, IdempotencyKey: document.IdempotencyKey}
	var meta responseMeta
	if c.authScheme == "ApiKey" {
		meta, err = c.requestMeta(ctx, http.MethodPost, path, payload, &response)
	} else {
		var out struct {
			Job hostIndexJob `json:"job"`
		}
		meta, err = c.invokeContract(ctx, documentManageCapability, struct {
			Operation string            `json:"operation"`
			SpaceUUID string            `json:"space_uuid"`
			Document  hostDocumentInput `json:"document"`
		}{"submit_document", document.SpaceID, payload}, &out)
		response = out.Job
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	job := response.toFramework()
	job.InvocationTraceID = meta.InvocationTraceID
	job.RequestID = meta.RequestID
	return job, nil
}

func (c *Client) DeleteKnowledgeDocument(ctx context.Context, input fwknowledge.DeleteDocumentInput) (*fwknowledge.KnowledgeIndexJob, error) {
	if strings.TrimSpace(input.SpaceID) == "" || strings.TrimSpace(input.DocumentID) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge space_id and document_id are required")
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/spaces/" + url.PathEscape(strings.TrimSpace(input.SpaceID)) + "/documents/" + url.PathEscape(strings.TrimSpace(input.DocumentID))
	if err := c.request(ctx, http.MethodDelete, path, nil, &response); err != nil {
		return nil, mapHostError(err)
	}
	return response.toFramework(), nil
}

func (c *Client) ReindexKnowledgeDocument(ctx context.Context, input fwknowledge.ReindexInput) (*fwknowledge.KnowledgeIndexJob, error) {
	if err := input.HostRebuild.Validate(); err != nil {
		return nil, err
	}
	space, err := uuid.Parse(input.SpaceID)
	if err != nil || space == uuid.Nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
	}
	path := "/api/v1/tenant/knowledge/spaces/" + space.String()
	if input.DocumentID != "" {
		document, err := uuid.Parse(input.DocumentID)
		if err != nil || document == uuid.Nil {
			return nil, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
		}
		input.DocumentID = document.String()
		path += "/documents/" + input.DocumentID
	}
	path += "/indexes:rebuild"
	var response hostIndexJob
	var meta responseMeta
	var requestErr error
	if c.authScheme == "ApiKey" {
		meta, requestErr = c.requestMeta(ctx, http.MethodPost, path, input.HostRebuild, &response)
	} else {
		operation := "rebuild_space"
		if input.DocumentID != "" {
			operation = "rebuild_document"
		}
		var out struct {
			Job hostIndexJob `json:"job"`
		}
		meta, requestErr = c.invokeContract(ctx, documentManageCapability, struct {
			Operation    string                               `json:"operation"`
			SpaceUUID    string                               `json:"space_uuid"`
			DocumentUUID string                               `json:"document_uuid,omitempty"`
			Rebuild      *fwknowledge.DocumentRebuildSettings `json:"rebuild"`
		}{operation, input.SpaceID, input.DocumentID, input.HostRebuild}, &out)
		response = out.Job
	}
	if requestErr != nil {
		return nil, mapHostError(requestErr)
	}
	jobID, err := uuid.Parse(response.JobUUID)
	if err != nil || jobID == uuid.Nil || response.Operation != "rebuild" || (response.Status != "queued" && response.Status != "running" && response.Status != "succeeded" && response.Status != "failed") || response.DocumentUUID != input.DocumentID {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidResponse, "knowledgeLab.workspace.invalidResponse")
	}
	job := response.toFramework()
	job.Operation = fwknowledge.IndexOperationReindex
	job.InvocationTraceID = meta.InvocationTraceID
	job.RequestID = meta.RequestID
	return job, nil
}

func (c *Client) GetKnowledgeIndexJob(ctx context.Context, input fwknowledge.IndexJobQuery) (*fwknowledge.KnowledgeIndexJob, error) {
	if strings.TrimSpace(input.JobID) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge job_id is required")
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/index-jobs/" + url.PathEscape(strings.TrimSpace(input.JobID))
	if !fwknowledge.SemanticUUID(input.JobID) {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	var meta responseMeta
	var err error
	if c.authScheme == "ApiKey" {
		meta, err = c.requestMeta(ctx, http.MethodGet, path, nil, &response)
	} else {
		var out struct {
			Job hostIndexJob `json:"job"`
		}
		meta, err = c.invokeContract(ctx, documentManageCapability, struct {
			Operation string `json:"operation"`
			JobUUID   string `json:"job_uuid"`
		}{"get_job", input.JobID}, &out)
		response = out.Job
	}
	if err != nil {
		return nil, mapHostError(err)
	}
	job := response.toFramework()
	job.InvocationTraceID = meta.InvocationTraceID
	job.RequestID = meta.RequestID
	return job, nil
}

type hostIndexJob struct {
	ArtifactCount      *int                                   `json:"artifact_count"`
	VectorCount        *int                                   `json:"vector_count"`
	BundleGeneration   string                                 `json:"bundle_generation"`
	RebuildMode        fwknowledge.DocumentRebuildMode        `json:"rebuild_mode"`
	DocumentCount      *int                                   `json:"document_count"`
	ProcessedDocuments *int                                   `json:"processed_documents"`
	DocumentConfigs    []fwknowledge.DocumentJobConfig        `json:"document_configs"`
	Priority           string                                 `json:"priority"`
	TraceID            string                                 `json:"trace_id"`
	EffectiveConfig    *fwknowledge.DocumentIngestionSnapshot `json:"effective_config"`
	RequestedConfig    *fwknowledge.DocumentIngestionSettings `json:"requested_config"`
	SnapshotChecksum   string                                 `json:"snapshot_checksum"`
	ChunkCount         *int                                   `json:"chunk_count"`
	StartedAt          time.Time                              `json:"started_at"`
	CompletedAt        time.Time                              `json:"completed_at"`

	JobUUID      string `json:"job_uuid"`
	SpaceUUID    string `json:"space_uuid"`
	DocumentUUID string `json:"document_uuid"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code"`
}

func (job hostIndexJob) toFramework() *fwknowledge.KnowledgeIndexJob {
	return &fwknowledge.KnowledgeIndexJob{ArtifactCount: job.ArtifactCount, VectorCount: job.VectorCount, BundleGeneration: job.BundleGeneration, RebuildMode: job.RebuildMode, DocumentCount: job.DocumentCount, ProcessedDocuments: job.ProcessedDocuments, DocumentConfigs: job.DocumentConfigs, Priority: job.Priority, TraceID: job.TraceID, EffectiveConfig: job.EffectiveConfig, RequestedConfig: job.RequestedConfig, SnapshotChecksum: job.SnapshotChecksum, ChunkCount: job.ChunkCount, StartedAt: job.StartedAt, FinishedAt: job.CompletedAt, JobID: strings.TrimSpace(job.JobUUID), SpaceID: strings.TrimSpace(job.SpaceUUID), DocumentID: strings.TrimSpace(job.DocumentUUID), Operation: strings.TrimSpace(job.Operation), Status: strings.TrimSpace(job.Status), ErrorCode: strings.TrimSpace(job.ErrorCode)}
}

func mapHostError(err error) error {
	var hostErr *HTTPError
	if !errors.As(err, &hostErr) {
		var typed *fwknowledge.Error
		if errors.As(err, &typed) {
			return err
		}
		return &fwknowledge.Error{Code: fwknowledge.CodeProviderUnavailable, StatusCode: 503, Message: "knowledge.host.request_failed", Cause: err}
	}
	code := fwknowledge.CodeProviderUnavailable
	switch hostErr.StatusCode {
	case http.StatusBadRequest:
		code = fwknowledge.CodeInvalidDocument
	case http.StatusUnauthorized:
		code = fwknowledge.CodeUnauthorized
	case http.StatusForbidden:
		code = fwknowledge.CodeForbidden
	case http.StatusNotFound:
		code = fwknowledge.CodeNotFound
	case http.StatusConflict:
		code = fwknowledge.CodeConflict
	case http.StatusPreconditionFailed:
		code = fwknowledge.CodeStrategyUnavailable
	case http.StatusTooManyRequests:
		code = fwknowledge.CodeRateLimited
	}
	if hostErr.ReasonCode != "" {
		code = fwknowledge.ErrorCode(hostErr.ReasonCode)
	}
	return &fwknowledge.Error{Code: code, StatusCode: hostErr.StatusCode, TraceID: hostErr.TraceID, RequestID: hostErr.RequestID, SafeDetails: hostErr.Details, Message: "knowledge.host.request_failed", Cause: err}

}

var _ fwknowledge.DelegatedClient = (*Client)(nil)

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

type stsToken struct{ client *sts.Client }

func (t stsToken) Token(ctx context.Context) (string, error) {
	token, err := t.client.Token(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (c *Client) GetKnowledgeCatalog(ctx context.Context) (*fwknowledge.KnowledgeCatalog, error) {
	var response struct {
		Catalog *fwknowledge.KnowledgeCatalog `json:"catalog"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/tenant/knowledge/catalog", nil, &response); err != nil {
		return nil, mapHostError(err)
	}
	if err := fwknowledge.ValidateHostCatalog(response.Catalog); err != nil {
		return nil, err
	}
	return response.Catalog, nil
}
func (c *Client) CreateKnowledgeSpace(ctx context.Context, input fwknowledge.CreateSpaceInput) (*fwknowledge.CreatedSpace, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var response struct {
		Item *fwknowledge.CreatedSpace `json:"item"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/v1/tenant/knowledge/spaces", input, &response); err != nil {
		return nil, mapHostError(err)
	}
	if err := fwknowledge.ValidateCreatedSpace(response.Item); err != nil {
		return nil, err
	}
	if response.Item.DepartmentUUID != input.DepartmentUUID || response.Item.StrategyKey != input.StrategyKey {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidResponse, "knowledge.provisioning.invalid_response")
	}
	return response.Item, nil
}

var _ fwknowledge.CatalogDelegatedClient = (*Client)(nil)
var _ fwknowledge.SpaceProvisioningDelegatedClient = (*Client)(nil)

func (c *Client) GetKnowledgeIndexJobChunks(ctx context.Context, input fwknowledge.IndexJobQuery) ([]fwknowledge.IndexJobChunk, error) {
	id, err := uuid.Parse(input.JobID)
	if err != nil || id == uuid.Nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
	}
	var response struct {
		Items []fwknowledge.IndexJobChunk `json:"items"`
	}
	var requestErr error
	if c.authScheme == "ApiKey" {
		requestErr = c.request(ctx, http.MethodGet, "/api/v1/tenant/knowledge/index-jobs/"+id.String()+"/chunks", nil, &response)
	} else {
		_, requestErr = c.invokeContract(ctx, documentManageCapability, struct {
			Operation string `json:"operation"`
			JobUUID   string `json:"job_uuid"`
		}{"get_job_chunks", id.String()}, &response)
	}
	if requestErr != nil {
		return nil, mapHostError(requestErr)
	}
	for _, chunk := range response.Items {
		if chunk.JobUUID != id.String() || chunk.UUID == "" || chunk.DocumentUUID == "" || chunk.SpaceUUID == "" || chunk.Checksum == "" {
			return nil, fwknowledge.NewError(fwknowledge.CodeInvalidResponse, "knowledgeLab.workspace.invalidResponse")
		}
	}
	return response.Items, nil
}
