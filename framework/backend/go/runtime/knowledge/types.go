package knowledge

import (
	"errors"
	"strings"
	"time"
)

const (
	ProviderModeLocal      = "local"
	ProviderModeDelegated  = "delegated"
	ProviderModeMock       = "mock"
	ProviderModeThirdParty = "third_party"

	VisibilityPrivate = "private"
	VisibilityTenant  = "tenant"
	VisibilityPlugin  = "plugin"
	VisibilityPublic  = "public"

	CallerTypeMember   = "member"
	CallerTypeCustomer = "customer"
	CallerTypeAgent    = "agent"
	CallerTypeSystem   = "system"

	OperationSearch   = "search"
	OperationRetrieve = "retrieve"
	OperationUpsert   = "upsert"
	OperationDelete   = "delete"
	OperationReindex  = "reindex"
	OperationHealth   = "health"
	OperationCatalog  = "catalog"

	IndexOperationUpsert  = "upsert"
	IndexOperationDelete  = "delete"
	IndexOperationReindex = "reindex"

	IndexStatusQueued    = "queued"
	IndexStatusRunning   = "running"
	IndexStatusSucceeded = "succeeded"
	IndexStatusFailed    = "failed"
	IndexStatusCancelled = "cancelled"

	DefaultQueryLimit = 10
	MaxQueryLimit     = 100
)

var (
	ErrQueryRequired      = errors.New("knowledge query is required")
	ErrDocumentIDRequired = errors.New("knowledge document_id is required")
	ErrSpaceIDRequired    = errors.New("knowledge space_id is required")
	ErrDocumentTitle      = errors.New("knowledge document title is required")
	ErrDocumentBody       = errors.New("knowledge document content or uri is required")
	ErrCitationRequired   = errors.New("knowledge citation is required")
	ErrSnippetText        = errors.New("knowledge snippet text is required")
)

type KnowledgeSpace struct {
	SpaceID                 string         `json:"space_id,omitempty"`
	SpaceName               string         `json:"space_name,omitempty"`
	PluginID                string         `json:"plugin_id,omitempty"`
	TenantUUID              string         `json:"tenant_uuid,omitempty"`
	DepartmentCode          string         `json:"department_code,omitempty"`
	Visibility              string         `json:"visibility,omitempty"`
	Status                  string         `json:"status,omitempty"`
	PolicyTemplateVersionID string         `json:"policy_template_version_id,omitempty"`
	IngestionProfileKey     string         `json:"ingestion_profile_key,omitempty"`
	IndexProfileKey         string         `json:"index_profile_key,omitempty"`
	RAGProfileKey           string         `json:"rag_profile_key,omitempty"`
	FeatureFlags            []string       `json:"feature_flags,omitempty"`
	Quotas                  map[string]any `json:"quotas,omitempty"`
	Locale                  string         `json:"locale,omitempty"`
	AgentIDs                []string       `json:"agent_ids,omitempty"`
	SkillIDs                []string       `json:"skill_ids,omitempty"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type ListSpacesInput struct {
	TenantUUID string         `json:"tenant_uuid,omitempty"`
	PluginID   string         `json:"plugin_id,omitempty"`
	Visibility string         `json:"visibility,omitempty"`
	Status     string         `json:"status,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	Filters    map[string]any `json:"filters,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
}

func (in ListSpacesInput) Normalized() ListSpacesInput {
	in.TenantUUID = strings.TrimSpace(in.TenantUUID)
	in.PluginID = strings.TrimSpace(in.PluginID)
	in.Visibility = normalizeVisibility(in.Visibility)
	in.Status = strings.TrimSpace(in.Status)
	in.TraceID = strings.TrimSpace(in.TraceID)
	if in.Limit <= 0 {
		in.Limit = DefaultQueryLimit
	}
	if in.Limit > MaxQueryLimit {
		in.Limit = MaxQueryLimit
	}
	if in.Filters == nil {
		in.Filters = map[string]any{}
	}
	return in
}

type KnowledgeDocument struct {
	DocumentID  string         `json:"document_id,omitempty"`
	SpaceID     string         `json:"space_id,omitempty"`
	Title       string         `json:"title,omitempty"`
	URI         string         `json:"uri,omitempty"`
	Content     string         `json:"content,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Checksum    string         `json:"checksum,omitempty"`
	Version     string         `json:"version,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Visibility  string         `json:"visibility,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	TenantUUID  string         `json:"tenant_uuid,omitempty"`
}

type KnowledgePosition map[string]any

type KnowledgeCitation struct {
	DocumentID  string            `json:"document_id,omitempty"`
	ChunkID     string            `json:"chunk_id,omitempty"`
	Title       string            `json:"title,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Version     string            `json:"version,omitempty"`
	Position    KnowledgePosition `json:"position,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	RetrievedAt time.Time         `json:"retrieved_at,omitempty"`
}

type KnowledgeChunk struct {
	ChunkID    string             `json:"chunk_id,omitempty"`
	DocumentID string             `json:"document_id,omitempty"`
	SpaceID    string             `json:"space_id,omitempty"`
	Text       string             `json:"text,omitempty"`
	Score      float64            `json:"score,omitempty"`
	Position   KnowledgePosition  `json:"position,omitempty"`
	Metadata   map[string]any     `json:"metadata,omitempty"`
	Citation   *KnowledgeCitation `json:"citation,omitempty"`
	TenantUUID string             `json:"tenant_uuid,omitempty"`
}

type KnowledgeQuery struct {
	Query      string         `json:"query,omitempty"`
	SpaceIDs   []string       `json:"space_ids,omitempty"`
	TenantUUID string         `json:"tenant_uuid,omitempty"`
	PluginID   string         `json:"plugin_id,omitempty"`
	AgentUUID  string         `json:"agent_uuid,omitempty"`
	SkillID    string         `json:"skill_id,omitempty"`
	CallerType string         `json:"caller_type,omitempty"`
	Locale     string         `json:"locale,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	MinScore   float64        `json:"min_score,omitempty"`
	Filters    map[string]any `json:"filters,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	Visibility string         `json:"visibility,omitempty"`
}

type KnowledgeSearchResult struct {
	QueryID     string              `json:"query_id,omitempty"`
	Provider    string              `json:"provider,omitempty"`
	SpaceID     string              `json:"space_id,omitempty"`
	Chunks      []KnowledgeChunk    `json:"chunks"`
	Citations   []KnowledgeCitation `json:"citations"`
	Total       int                 `json:"total"`
	Diagnostics map[string]any      `json:"diagnostics,omitempty"`
	TraceID     string              `json:"trace_id,omitempty"`
}

type KnowledgeIndexJob struct {
	JobID      string    `json:"job_id,omitempty"`
	SpaceID    string    `json:"space_id,omitempty"`
	DocumentID string    `json:"document_id,omitempty"`
	Operation  string    `json:"operation,omitempty"`
	Status     string    `json:"status,omitempty"`
	ErrorCode  string    `json:"error_code,omitempty"`
	Message    string    `json:"message,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

type DeleteDocumentInput struct {
	SpaceID    string `json:"space_id,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	TenantUUID string `json:"tenant_uuid,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

type ReindexInput struct {
	SpaceID    string `json:"space_id,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	TenantUUID string `json:"tenant_uuid,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

type StrategyDependency struct {
	Index   []string `json:"index,omitempty"`
	Runtime []string `json:"runtime,omitempty"`
	Assets  []string `json:"assets,omitempty"`
}

type KnowledgeScene struct {
	Key                    string             `json:"key"`
	Label                  string             `json:"label"`
	Description            string             `json:"description,omitempty"`
	DefaultStrategyPackage string             `json:"default_strategy_package,omitempty"`
	DefaultBundle          string             `json:"default_bundle,omitempty"`
	AllowedBundles         []string           `json:"allowed_bundles,omitempty"`
	Dependencies           StrategyDependency `json:"dependencies,omitempty"`
	Metadata               map[string]any     `json:"metadata,omitempty"`
}

type StrategyPackage struct {
	Key                   string             `json:"key"`
	Label                 string             `json:"label"`
	Summary               string             `json:"summary,omitempty"`
	DisplayLabel          string             `json:"display_label,omitempty"`
	UseCase               string             `json:"use_case,omitempty"`
	NotFor                string             `json:"not_for,omitempty"`
	RecommendedProfileKey string             `json:"recommended_profile_key,omitempty"`
	RecommendedScenes     []string           `json:"recommended_scenes,omitempty"`
	Dependencies          StrategyDependency `json:"dependencies,omitempty"`
	Metadata              map[string]any     `json:"metadata,omitempty"`
}

type StrategyBundle struct {
	Key           string         `json:"key"`
	Label         string         `json:"label"`
	Description   string         `json:"description,omitempty"`
	Prerequisites []string       `json:"prerequisites,omitempty"`
	Guardrails    map[string]any `json:"guardrails,omitempty"`
}

type KnowledgeCatalog struct {
	Version          string            `json:"version,omitempty"`
	Source           string            `json:"source,omitempty"`
	Scenes           []KnowledgeScene  `json:"scenes"`
	StrategyPackages []StrategyPackage `json:"strategy_packages"`
	StrategyBundles  []StrategyBundle  `json:"strategy_bundles,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
}

func (q KnowledgeQuery) Normalized() KnowledgeQuery {
	q.Query = strings.TrimSpace(q.Query)
	q.TenantUUID = strings.TrimSpace(q.TenantUUID)
	q.PluginID = strings.TrimSpace(q.PluginID)
	q.AgentUUID = strings.TrimSpace(q.AgentUUID)
	q.SkillID = strings.TrimSpace(q.SkillID)
	q.CallerType = strings.ToLower(strings.TrimSpace(q.CallerType))
	q.Locale = strings.TrimSpace(q.Locale)
	q.TraceID = strings.TrimSpace(q.TraceID)
	q.Visibility = normalizeVisibility(q.Visibility)
	q.SpaceIDs = trimStrings(q.SpaceIDs)
	q.Tags = trimStrings(q.Tags)
	if q.Limit <= 0 {
		q.Limit = DefaultQueryLimit
	}
	if q.Limit > MaxQueryLimit {
		q.Limit = MaxQueryLimit
	}
	if q.MinScore < 0 {
		q.MinScore = 0
	}
	if q.MinScore > 1 {
		q.MinScore = 1
	}
	if q.Filters == nil {
		q.Filters = map[string]any{}
	}
	return q
}

func (q KnowledgeQuery) Validate(requireTenant bool) error {
	q = q.Normalized()
	if q.Query == "" {
		return NewError(CodeInvalidDocument, "knowledge query is required")
	}
	if requireTenant && q.TenantUUID == "" {
		return NewError(CodeTenantRequired, "knowledge tenant_uuid is required")
	}
	return nil
}

func (d KnowledgeDocument) Normalized() KnowledgeDocument {
	d.DocumentID = strings.TrimSpace(d.DocumentID)
	d.SpaceID = strings.TrimSpace(d.SpaceID)
	d.Title = strings.TrimSpace(d.Title)
	d.URI = strings.TrimSpace(d.URI)
	d.Content = strings.TrimSpace(d.Content)
	d.ContentType = strings.TrimSpace(d.ContentType)
	d.Checksum = strings.TrimSpace(d.Checksum)
	d.Version = strings.TrimSpace(d.Version)
	d.Visibility = normalizeVisibility(d.Visibility)
	d.TenantUUID = strings.TrimSpace(d.TenantUUID)
	d.Tags = trimStrings(d.Tags)
	if d.Metadata == nil {
		d.Metadata = map[string]any{}
	}
	return d
}

func (d KnowledgeDocument) Validate() error {
	d = d.Normalized()
	if d.DocumentID == "" {
		return NewError(CodeInvalidDocument, ErrDocumentIDRequired.Error())
	}
	if d.SpaceID == "" {
		return NewError(CodeInvalidDocument, ErrSpaceIDRequired.Error())
	}
	if d.Title == "" {
		return NewError(CodeInvalidDocument, ErrDocumentTitle.Error())
	}
	if d.Content == "" && d.URI == "" {
		return NewError(CodeInvalidDocument, ErrDocumentBody.Error())
	}
	return nil
}

func (c KnowledgeChunk) Validate() error {
	if strings.TrimSpace(c.Text) == "" {
		return NewError(CodeInvalidDocument, ErrSnippetText.Error())
	}
	if c.Citation == nil || strings.TrimSpace(c.Citation.DocumentID) == "" || strings.TrimSpace(c.Citation.Title) == "" {
		return NewError(CodeInvalidDocument, ErrCitationRequired.Error())
	}
	return nil
}

func TenantRequiredForVisibility(visibility string) bool {
	switch normalizeVisibility(visibility) {
	case VisibilityTenant, VisibilityPrivate:
		return true
	default:
		return false
	}
}

func normalizeVisibility(visibility string) string {
	switch strings.ToLower(strings.TrimSpace(visibility)) {
	case VisibilityPrivate, VisibilityTenant, VisibilityPlugin, VisibilityPublic:
		return strings.ToLower(strings.TrimSpace(visibility))
	default:
		return ""
	}
}

func trimStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
