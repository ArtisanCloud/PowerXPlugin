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
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/sts"
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
	baseURL string
	http    *http.Client
	token   tokenProvider
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

func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	if c == nil || c.http == nil || c.token == nil {
		return errors.New("powerx knowledge client is not configured")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	token, err := c.token.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			ReasonCode string `json:"reason_code"`
			ErrorCode  string `json:"error_code"`
		}
		_ = json.Unmarshal(raw, &envelope)
		reason := strings.TrimSpace(envelope.ReasonCode)
		if reason == "" {
			reason = strings.TrimSpace(envelope.ErrorCode)
		}
		return &HTTPError{StatusCode: resp.StatusCode, ReasonCode: reason, Body: string(raw)}
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return errors.New("powerx knowledge response missing data")
	}
	return json.Unmarshal(envelope.Data, output)
}

type HTTPError struct {
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
		fwknowledge.OperationRetrieve, fwknowledge.OperationSearch, fwknowledge.OperationUpsert,
		fwknowledge.OperationDelete, fwknowledge.OperationReindex, fwknowledge.OperationHealth)
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
	document, err := fwknowledge.ValidateDocument(document)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(document.Content) == "" || strings.TrimSpace(document.ContentType) == "" || strings.TrimSpace(document.Checksum) == "" || strings.TrimSpace(document.Version) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge Host document requires content, content_type, checksum, and version")
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/spaces/" + url.PathEscape(document.SpaceID) + "/documents"
	payload := map[string]any{"title": document.Title, "uri": document.URI, "content": document.Content, "content_type": document.ContentType, "checksum": document.Checksum, "version": document.Version, "tags": document.Tags}
	if err := c.request(ctx, http.MethodPost, path, payload, &response); err != nil {
		return nil, mapHostError(err)
	}
	return response.toFramework(), nil
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
	if strings.TrimSpace(input.SpaceID) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge space_id is required")
	}
	if strings.TrimSpace(input.DocumentID) != "" {
		return nil, fwknowledge.Unsupported(fwknowledge.OperationReindex)
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/spaces/" + url.PathEscape(strings.TrimSpace(input.SpaceID)) + "/indexes:rebuild"
	if err := c.request(ctx, http.MethodPost, path, nil, &response); err != nil {
		return nil, mapHostError(err)
	}
	job := response.toFramework()
	job.Operation = fwknowledge.IndexOperationReindex
	return job, nil
}

func (c *Client) GetKnowledgeIndexJob(ctx context.Context, input fwknowledge.IndexJobQuery) (*fwknowledge.KnowledgeIndexJob, error) {
	if strings.TrimSpace(input.JobID) == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge job_id is required")
	}
	var response hostIndexJob
	path := "/api/v1/tenant/knowledge/index-jobs/" + url.PathEscape(strings.TrimSpace(input.JobID))
	if err := c.request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, mapHostError(err)
	}
	return response.toFramework(), nil
}

type hostIndexJob struct {
	JobUUID      string `json:"job_uuid"`
	SpaceUUID    string `json:"space_uuid"`
	DocumentUUID string `json:"document_uuid"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code"`
}

func (job hostIndexJob) toFramework() *fwknowledge.KnowledgeIndexJob {
	return &fwknowledge.KnowledgeIndexJob{JobID: strings.TrimSpace(job.JobUUID), SpaceID: strings.TrimSpace(job.SpaceUUID), DocumentID: strings.TrimSpace(job.DocumentUUID), Operation: strings.TrimSpace(job.Operation), Status: strings.TrimSpace(job.Status), ErrorCode: strings.TrimSpace(job.ErrorCode)}
}

func mapHostError(err error) error {
	var hostErr *HTTPError
	if !errors.As(err, &hostErr) {
		return err
	}
	switch hostErr.StatusCode {
	case http.StatusBadRequest:
		return fwknowledge.WrapError(fwknowledge.CodeInvalidDocument, "knowledge Host request is invalid", err)
	case http.StatusUnauthorized:
		return fwknowledge.WrapError(fwknowledge.CodeUnauthorized, "knowledge Host request was rejected", err)
	case http.StatusForbidden:
		return fwknowledge.WrapError(fwknowledge.CodeForbidden, "knowledge Host capability was forbidden", err)
	case http.StatusNotFound:
		return fwknowledge.WrapError(fwknowledge.CodeNotFound, "knowledge Host object was not found", err)
	case http.StatusConflict:
		return fwknowledge.WrapError(fwknowledge.CodeConflict, "knowledge Host index operation conflicts", err)
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return fwknowledge.WrapError(fwknowledge.CodeProviderUnavailable, "knowledge Host dependency is unavailable", err)
	default:
		return fwknowledge.WrapError(fwknowledge.CodeProviderUnavailable, "knowledge Host request failed", err)
	}
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
