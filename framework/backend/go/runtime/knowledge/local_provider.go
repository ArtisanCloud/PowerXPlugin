package knowledge

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type LocalProviderConfig struct {
	Name          string
	RequireTenant bool
}

type LocalProvider struct {
	mu            sync.RWMutex
	name          string
	requireTenant bool
	documents     map[string]KnowledgeDocument
	jobs          map[string]KnowledgeIndexJob
}

func NewLocalProvider(cfg LocalProviderConfig) *LocalProvider {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = "local"
	}
	return &LocalProvider{
		name:          name,
		requireTenant: cfg.RequireTenant,
		documents:     map[string]KnowledgeDocument{},
		jobs:          map[string]KnowledgeIndexJob{},
	}
}

func (p *LocalProvider) Name() string { return p.name }

func (p *LocalProvider) Mode() string { return ProviderModeLocal }

func (p *LocalProvider) Capabilities(context.Context) ProviderCapabilities {
	return BasicCapabilities(p.name, ProviderModeLocal, OperationSearch, OperationUpsert, OperationDelete, OperationReindex, OperationHealth, OperationCatalog)
}

func (p *LocalProvider) Catalog(context.Context) (*KnowledgeCatalog, error) {
	return DefaultKnowledgeCatalog("framework_local"), nil
}

func (p *LocalProvider) ListSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error) {
	_ = ctx
	input = input.Normalized()
	if p.requireTenant && input.TenantUUID == "" {
		return nil, NewError(CodeTenantRequired, "knowledge tenant_uuid is required")
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	bySpace := map[string]KnowledgeSpace{}
	for _, document := range p.documents {
		if input.TenantUUID != "" && document.TenantUUID != "" && document.TenantUUID != input.TenantUUID {
			continue
		}
		if input.Visibility != "" && document.Visibility != "" && document.Visibility != input.Visibility {
			continue
		}
		spaceID := strings.TrimSpace(document.SpaceID)
		if spaceID == "" {
			continue
		}
		space, exists := bySpace[spaceID]
		if !exists {
			space = KnowledgeSpace{
				SpaceID:        spaceID,
				SpaceName:      firstNonEmpty(document.MetadataString("space_name"), spaceID),
				TenantUUID:     document.TenantUUID,
				DepartmentCode: document.MetadataString("department_code"),
				Visibility:     document.Visibility,
				Status:         firstNonEmpty(document.MetadataString("status"), "active"),
				FeatureFlags:   []string{},
				Quotas:         map[string]any{},
				Metadata:       map[string]any{"document_count": 0},
			}
		}
		space.Metadata["document_count"] = intFromAny(space.Metadata["document_count"]) + 1
		bySpace[spaceID] = space
	}
	spaces := make([]KnowledgeSpace, 0, len(bySpace))
	for _, space := range bySpace {
		spaces = append(spaces, space)
	}
	sort.SliceStable(spaces, func(i, j int) bool {
		return strings.ToLower(spaces[i].SpaceName) < strings.ToLower(spaces[j].SpaceName)
	})
	if len(spaces) > input.Limit {
		spaces = spaces[:input.Limit]
	}
	return spaces, nil
}

func (p *LocalProvider) Search(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error) {
	_ = ctx
	started := time.Now()
	query = query.Normalized()
	if err := query.Validate(p.requireTenant || TenantRequiredForVisibility(query.Visibility)); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	lowerQuery := strings.ToLower(query.Query)
	chunks := make([]KnowledgeChunk, 0)
	for _, document := range p.documents {
		if !matchesQueryScope(document, query) {
			continue
		}
		score := localScore(document, lowerQuery)
		if score < query.MinScore || score == 0 {
			continue
		}
		text := document.Content
		if len(text) > 500 {
			text = text[:500]
		}
		citation := &KnowledgeCitation{
			DocumentID:  document.DocumentID,
			ChunkID:     document.DocumentID + ":chunk:0",
			Title:       document.Title,
			URI:         document.URI,
			Version:     document.Version,
			Provider:    p.name,
			RetrievedAt: time.Now().UTC(),
		}
		chunks = append(chunks, KnowledgeChunk{
			ChunkID:    citation.ChunkID,
			DocumentID: document.DocumentID,
			SpaceID:    document.SpaceID,
			Text:       text,
			Score:      score,
			Metadata:   RedactMap(document.Metadata),
			Citation:   citation,
			TenantUUID: document.TenantUUID,
		})
	}
	sort.SliceStable(chunks, func(i, j int) bool { return chunks[i].Score > chunks[j].Score })
	if len(chunks) > query.Limit {
		chunks = chunks[:query.Limit]
	}
	result := &KnowledgeSearchResult{
		Provider:    p.name,
		Chunks:      chunks,
		Diagnostics: QueryDiagnostics(p, OperationSearch, query, started),
		TraceID:     query.TraceID,
	}
	return NormalizeSearchResult(p, query, result)
}

func (p *LocalProvider) UpsertDocument(ctx context.Context, document KnowledgeDocument) (*KnowledgeIndexJob, error) {
	_ = ctx
	document, err := ValidateDocument(document)
	if err != nil {
		return nil, err
	}
	if p.requireTenant && document.TenantUUID == "" {
		return nil, NewError(CodeTenantRequired, "knowledge tenant_uuid is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if document.DocumentID == "" {
		document.DocumentID = uuid.NewString()
	}
	p.documents[documentKey(document.SpaceID, document.DocumentID)] = document
	job := CompletedIndexJob(IndexOperationUpsert, document)
	p.jobs[job.JobID] = *job
	return job, nil
}

func (p *LocalProvider) DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*KnowledgeIndexJob, error) {
	_ = ctx
	input.SpaceID = strings.TrimSpace(input.SpaceID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	if input.SpaceID == "" {
		return nil, NewError(CodeInvalidDocument, ErrSpaceIDRequired.Error())
	}
	if input.DocumentID == "" {
		return nil, NewError(CodeInvalidDocument, ErrDocumentIDRequired.Error())
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.documents, documentKey(input.SpaceID, input.DocumentID))
	now := time.Now().UTC()
	job := &KnowledgeIndexJob{JobID: uuid.NewString(), SpaceID: input.SpaceID, DocumentID: input.DocumentID, Operation: IndexOperationDelete, Status: IndexStatusSucceeded, StartedAt: now, FinishedAt: now}
	p.jobs[job.JobID] = *job
	return job, nil
}

func (p *LocalProvider) Reindex(ctx context.Context, input ReindexInput) (*KnowledgeIndexJob, error) {
	_ = ctx
	input.SpaceID = strings.TrimSpace(input.SpaceID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	if input.SpaceID == "" {
		return nil, NewError(CodeInvalidDocument, ErrSpaceIDRequired.Error())
	}
	now := time.Now().UTC()
	job := &KnowledgeIndexJob{JobID: uuid.NewString(), SpaceID: input.SpaceID, DocumentID: input.DocumentID, Operation: IndexOperationReindex, Status: IndexStatusSucceeded, StartedAt: now, FinishedAt: now}
	p.mu.Lock()
	p.jobs[job.JobID] = *job
	p.mu.Unlock()
	return job, nil
}

func (p *LocalProvider) GetIndexJob(_ context.Context, input IndexJobQuery) (*KnowledgeIndexJob, error) {
	input.JobID = strings.TrimSpace(input.JobID)
	if input.JobID == "" {
		return nil, NewError(CodeInvalidDocument, "knowledge job_id is required")
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	job, ok := p.jobs[input.JobID]
	if !ok {
		return nil, NewError(CodeNotFound, "knowledge index job not found")
	}
	return &job, nil
}

func matchesQueryScope(document KnowledgeDocument, query KnowledgeQuery) bool {
	if query.TenantUUID != "" && document.TenantUUID != "" && query.TenantUUID != document.TenantUUID {
		return false
	}
	if len(query.SpaceIDs) > 0 {
		found := false
		for _, spaceID := range query.SpaceIDs {
			if document.SpaceID == spaceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, tag := range query.Tags {
		if !hasTag(document.Tags, tag) {
			return false
		}
	}
	return true
}

func localScore(document KnowledgeDocument, lowerQuery string) float64 {
	haystack := strings.ToLower(strings.Join([]string{document.Title, document.Content, strings.Join(document.Tags, " ")}, " "))
	if !strings.Contains(haystack, lowerQuery) {
		return 0
	}
	if strings.Contains(strings.ToLower(document.Title), lowerQuery) {
		return 1
	}
	return 0.75
}

func hasTag(tags []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func documentKey(spaceID, documentID string) string {
	return strings.TrimSpace(spaceID) + "/" + strings.TrimSpace(documentID)
}

func (d KnowledgeDocument) MetadataString(key string) string {
	if d.Metadata == nil {
		return ""
	}
	if value, ok := d.Metadata[key]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
