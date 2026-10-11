package knowledge

import (
	"context"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

const (
	ContentOverviewQuerySchema  = "powerx.knowledge.content-overview-query/v1"
	ContentOverviewResultSchema = "powerx.knowledge.content-overview/v1"
	ContentOverviewCapability   = "com.corex.knowledge.content_overview.service_read"
	OperationContentOverview    = "content_overview"
	ContentAuthorizationScope   = "current_credential_tenant_published_corpus"
	ContentDerivedNotProduced   = "KNOWLEDGE_CONTENT_DERIVED_NOT_PRODUCED"
)

type ContentOverviewQuery struct {
	Schema                   string `json:"schema"`
	SpaceUUID                string `json:"space_uuid"`
	Mode                     string `json:"mode"`
	RequiredCorpusGeneration string `json:"required_corpus_generation,omitempty"`
	PageSize                 *int   `json:"page_size,omitempty"`
	PreviewCodepoints        *int   `json:"preview_codepoints,omitempty"`
	Cursor                   string `json:"cursor,omitempty"`
}
type ContentPreview struct {
	Kind         string `json:"kind"`
	Text         string `json:"text"`
	PositionUnit string `json:"position_unit"`
	CharStart    int    `json:"char_start"`
	CharEnd      int    `json:"char_end"`
	Truncated    bool   `json:"truncated"`
}
type ContentOverviewItem struct {
	DocumentUUID     string            `json:"document_uuid"`
	Title            string            `json:"title"`
	URI              string            `json:"uri"`
	DocumentVersion  string            `json:"document_version"`
	BundleGeneration string            `json:"bundle_generation"`
	ChunkUUID        string            `json:"chunk_uuid"`
	ArtifactUUID     string            `json:"artifact_uuid"`
	SourceRef        SemanticSourceRef `json:"source_ref"`
	Preview          ContentPreview    `json:"preview"`
}
type ContentOverviewPagination struct {
	PageSize   int     `json:"page_size"`
	Total      int     `json:"total"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}
type ContentDerivedStatus struct {
	Status     string `json:"status"`
	ReasonCode string `json:"reason_code"`
}
type ContentOverviewResult struct {
	Schema               string                    `json:"schema"`
	SpaceUUID            string                    `json:"space_uuid"`
	Mode                 string                    `json:"mode"`
	AuthorizationScope   string                    `json:"authorization_scope"`
	Generation           SemanticGeneration        `json:"generation"`
	Items                []ContentOverviewItem     `json:"items"`
	Pagination           ContentOverviewPagination `json:"pagination"`
	Overview             ContentDerivedStatus      `json:"overview"`
	Themes               ContentDerivedStatus      `json:"themes"`
	RecommendedQuestions ContentDerivedStatus      `json:"recommended_questions"`
	TraceID              string                    `json:"trace_id"`
	RequestID            string                    `json:"request_id,omitempty"`
	InvocationTraceID    string                    `json:"invocation_trace_id,omitempty"`
}

// Optional extension: adding the Host contract does not widen existing Local
// stores, clients or mocks. A Local implementation must be supplied by its owner.
type ContentOverviewService interface {
	ListContentOverview(context.Context, ContentOverviewQuery) (*ContentOverviewResult, error)
}

func ContentOverviewProvider(provider KnowledgeProvider) (ContentOverviewService, error) {
	service, ok := provider.(ContentOverviewService)
	if !ok || service == nil {
		return nil, Unsupported(OperationContentOverview)
	}
	return service, nil
}
func (p *DelegatedProvider) ListContentOverview(ctx context.Context, query ContentOverviewQuery) (*ContentOverviewResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if err := RequireCapability(p.Capabilities(ctx), OperationContentOverview); err != nil {
		return nil, err
	}
	client, ok := p.client.(ContentOverviewService)
	if !ok || client == nil {
		return nil, Unsupported(OperationContentOverview)
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := client.ListContentOverview(ctx, query)
	if err != nil {
		return nil, p.mapError(OperationContentOverview, "", err)
	}
	if result == nil {
		return nil, NewError(CodeInvalidResponse, "knowledgeContent.invalidResponse")
	}
	if err := result.Validate(query); err != nil {
		return nil, err
	}
	return result, nil
}
func (query ContentOverviewQuery) Validate() error {
	if query.Schema != ContentOverviewQuerySchema || !SemanticUUID(query.SpaceUUID) || !validMode(query.Mode) || query.RequiredCorpusGeneration != "" && !SemanticUUID(query.RequiredCorpusGeneration) || query.PageSize != nil && (*query.PageSize < 1 || *query.PageSize > 100) || query.PreviewCodepoints != nil && (*query.PreviewCodepoints < 0 || *query.PreviewCodepoints > 2000) || len(query.Cursor) > 4096 || strings.TrimSpace(query.Cursor) != query.Cursor {
		return NewError(CodeInvalidArgument, "knowledgeContent.invalidRequest")
	}
	return nil
}
func (result ContentOverviewResult) Validate(query ContentOverviewQuery) error {
	invalid := func() error { return NewError(CodeInvalidResponse, "knowledgeContent.invalidResponse") }
	g, page := result.Generation, result.Pagination
	size, previewLimit := 20, 320
	if query.PageSize != nil {
		size = *query.PageSize
	}
	if query.PreviewCodepoints != nil {
		previewLimit = *query.PreviewCodepoints
	}
	if result.Schema != ContentOverviewResultSchema || result.SpaceUUID != query.SpaceUUID || result.Mode != query.Mode || result.AuthorizationScope != ContentAuthorizationScope || g.SpaceUUID != query.SpaceUUID || !SemanticUUID(g.ConfigurationGeneration) || !SemanticUUID(g.CorpusGeneration) || !SemanticUUID(g.EmbeddingProfile.UUID) || g.EmbeddingProfile.Version < 1 || g.Dimensions < 1 || g.ModelKey == "" || g.ModelRevision == "" || g.DistanceMetric != "cosine" || query.RequiredCorpusGeneration != "" && g.CorpusGeneration != query.RequiredCorpusGeneration || result.Items == nil || page.PageSize != size || page.Total < 0 || len(result.Items) > size || len(result.Items) > page.Total || page.Total > 0 && len(result.Items) == 0 {
		return invalid()
	}
	if page.HasMore {
		if page.NextCursor == nil || *page.NextCursor == "" || len(*page.NextCursor) > 4096 || len(result.Items) == 0 || len(result.Items) >= page.Total {
			return invalid()
		}
	} else if page.NextCursor != nil || query.Cursor == "" && len(result.Items) != page.Total {
		return invalid()
	}
	for _, status := range []ContentDerivedStatus{result.Overview, result.Themes, result.RecommendedQuestions} {
		if status.Status != "not_available" || status.ReasonCode != ContentDerivedNotProduced {
			return invalid()
		}
	}
	seen := map[string]bool{}
	for _, item := range result.Items {
		source, preview := item.SourceRef, item.Preview
		if !SemanticUUID(item.DocumentUUID) || seen[item.DocumentUUID] || strings.TrimSpace(item.Title) == "" || item.DocumentVersion == "" || !SemanticUUID(item.BundleGeneration) || !SemanticUUID(item.ChunkUUID) || !SemanticUUID(item.ArtifactUUID) || !SemanticUUID(source.SourceUUID) || source.Version == "" || source.PositionUnit != "unicode_codepoint" || source.CharStart < 0 || source.CharEnd <= source.CharStart || len(source.Checksum) != 64 || source.Kind != "raw_document" && source.Kind != "cleaned_document" || preview.Kind != "published_source_chunk" || preview.PositionUnit != "unicode_codepoint" || !utf8.ValidString(preview.Text) || preview.CharStart != source.CharStart || preview.CharEnd < preview.CharStart || preview.CharEnd > source.CharEnd || preview.CharEnd-preview.CharStart != utf8.RuneCountInString(preview.Text) || utf8.RuneCountInString(preview.Text) > previewLimit || preview.Truncated != (preview.CharEnd < source.CharEnd) {
			return invalid()
		}
		if _, err := hex.DecodeString(source.Checksum); err != nil {
			return invalid()
		}
		seen[item.DocumentUUID] = true
	}
	return nil
}
