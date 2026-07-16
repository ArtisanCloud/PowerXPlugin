package knowledge

import (
	"context"
	"testing"
)

type recordingProvider struct {
	*MockProvider
	lastQuery KnowledgeQuery
}

func (p *recordingProvider) Search(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error) {
	p.lastQuery = query
	return p.MockProvider.Search(ctx, query)
}

func TestRAGRetrieverSuccess(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund policy")
	provider := NewMockProvider()
	provider.SearchResult = FixtureSearchResult("mock", doc, "refund policy")
	retriever := NewRAGRetriever(provider)
	result, err := retriever.Retrieve(context.Background(), RAGContext{TenantUUID: "tenant-a", Visibility: VisibilityTenant}, "refund")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if result.Total != 1 || len(result.Citations) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRAGRetrieverPropagatesCallerContext(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund policy")
	provider := &recordingProvider{MockProvider: NewMockProvider()}
	provider.SearchResult = FixtureSearchResult("mock", doc, "refund policy")
	retriever := NewRAGRetriever(provider)
	_, err := retriever.Retrieve(context.Background(), RAGContext{
		TenantUUID: "tenant-a",
		PluginID:   "com.powerx.plugins.base",
		AgentUUID:  "agent-a",
		SkillID:    "skill-a",
		CallerType: CallerTypeAgent,
		Locale:     "zh-CN",
		TraceID:    "trace-a",
		Visibility: VisibilityTenant,
	}, "refund")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if provider.lastQuery.TenantUUID != "tenant-a" ||
		provider.lastQuery.PluginID != "com.powerx.plugins.base" ||
		provider.lastQuery.AgentUUID != "agent-a" ||
		provider.lastQuery.SkillID != "skill-a" ||
		provider.lastQuery.CallerType != CallerTypeAgent ||
		provider.lastQuery.Locale != "zh-CN" ||
		provider.lastQuery.TraceID != "trace-a" ||
		provider.lastQuery.Visibility != VisibilityTenant {
		t.Fatalf("caller context was not propagated: %+v", provider.lastQuery)
	}
}

func TestRAGRetrieverTenantRequired(t *testing.T) {
	retriever := NewRAGRetriever(NewLocalProvider(LocalProviderConfig{RequireTenant: true}))
	_, err := retriever.Retrieve(context.Background(), RAGContext{Visibility: VisibilityTenant}, "refund")
	if CodeOf(err) != CodeTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
}
