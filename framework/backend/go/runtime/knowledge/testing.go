package knowledge

import "context"

type MockProvider struct {
	ProviderName  string
	ProviderMode  string
	Caps          ProviderCapabilities
	Spaces        []KnowledgeSpace
	ListSpacesErr error
	SearchResult  *KnowledgeSearchResult
	SearchErr     error
	UpsertErr     error
	DeleteErr     error
	ReindexErr    error
	IndexJobErr   error
	IndexJob      *KnowledgeIndexJob
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		ProviderName: "mock",
		ProviderMode: ProviderModeMock,
		Caps:         BasicCapabilities("mock", ProviderModeMock, OperationSearch, OperationUpsert, OperationDelete, OperationReindex, OperationHealth, OperationCatalog),
	}
}

func (p *MockProvider) Name() string {
	if p.ProviderName == "" {
		return "mock"
	}
	return p.ProviderName
}

func (p *MockProvider) Mode() string {
	if p.ProviderMode == "" {
		return ProviderModeMock
	}
	return p.ProviderMode
}

func (p *MockProvider) Capabilities(context.Context) ProviderCapabilities {
	if p.Caps.Provider == "" {
		p.Caps = BasicCapabilities(p.Name(), p.Mode(), OperationSearch)
	}
	return p.Caps
}

func (p *MockProvider) ListSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error) {
	_ = ctx
	if p.ListSpacesErr != nil {
		return nil, p.ListSpacesErr
	}
	if p.Spaces == nil {
		return []KnowledgeSpace{}, nil
	}
	return append([]KnowledgeSpace(nil), p.Spaces...), nil
}

func (p *MockProvider) Catalog(context.Context) (*KnowledgeCatalog, error) {
	return DefaultKnowledgeCatalog("framework_mock"), nil
}

func (p *MockProvider) Search(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error) {
	_ = ctx
	if p.SearchErr != nil {
		return nil, p.SearchErr
	}
	if p.SearchResult == nil {
		return NormalizeSearchResult(p, query, &KnowledgeSearchResult{Provider: p.Name(), Chunks: []KnowledgeChunk{}, Citations: []KnowledgeCitation{}, Total: 0, TraceID: query.TraceID})
	}
	return NormalizeSearchResult(p, query, p.SearchResult)
}

func (p *MockProvider) UpsertDocument(ctx context.Context, document KnowledgeDocument) (*KnowledgeIndexJob, error) {
	_ = ctx
	if p.UpsertErr != nil {
		return nil, p.UpsertErr
	}
	document, err := ValidateDocument(document)
	if err != nil {
		return nil, err
	}
	return CompletedIndexJob(IndexOperationUpsert, document), nil
}

func (p *MockProvider) DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*KnowledgeIndexJob, error) {
	_ = ctx
	if p.DeleteErr != nil {
		return nil, p.DeleteErr
	}
	return &KnowledgeIndexJob{JobID: "mock-delete", SpaceID: input.SpaceID, DocumentID: input.DocumentID, Operation: IndexOperationDelete, Status: IndexStatusSucceeded}, nil
}

func (p *MockProvider) Reindex(ctx context.Context, input ReindexInput) (*KnowledgeIndexJob, error) {
	_ = ctx
	if p.ReindexErr != nil {
		return nil, p.ReindexErr
	}
	return &KnowledgeIndexJob{JobID: "mock-reindex", SpaceID: input.SpaceID, DocumentID: input.DocumentID, Operation: IndexOperationReindex, Status: IndexStatusSucceeded}, nil
}

func (p *MockProvider) GetIndexJob(context.Context, IndexJobQuery) (*KnowledgeIndexJob, error) {
	if p.IndexJobErr != nil {
		return nil, p.IndexJobErr
	}
	if p.IndexJob == nil {
		return nil, NewError(CodeNotFound, "knowledge index job not found")
	}
	copy := *p.IndexJob
	return &copy, nil
}

func FixtureDocument(spaceID, documentID, title, content string) KnowledgeDocument {
	return KnowledgeDocument{
		SpaceID:     spaceID,
		DocumentID:  documentID,
		Title:       title,
		Content:     content,
		ContentType: "text/markdown",
		Visibility:  VisibilityTenant,
	}
}

func FixtureSearchResult(provider string, document KnowledgeDocument, text string) *KnowledgeSearchResult {
	citation := &KnowledgeCitation{DocumentID: document.DocumentID, ChunkID: document.DocumentID + ":chunk:0", Title: document.Title, URI: document.URI, Version: document.Version, Provider: provider}
	return &KnowledgeSearchResult{
		Provider: provider,
		Chunks: []KnowledgeChunk{{
			ChunkID:    citation.ChunkID,
			DocumentID: document.DocumentID,
			SpaceID:    document.SpaceID,
			Text:       text,
			Score:      1,
			Citation:   citation,
			TenantUUID: document.TenantUUID,
		}},
	}
}
