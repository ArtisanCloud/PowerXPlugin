package knowledge

import (
	"context"
	"testing"
)

func TestProviderCapabilities(t *testing.T) {
	caps := BasicCapabilities("local", ProviderModeLocal, OperationSearch)
	if err := RequireCapability(caps, OperationSearch); err != nil {
		t.Fatalf("search should be supported: %v", err)
	}
	if err := RequireCapability(caps, OperationDelete); CodeOf(err) != CodeUnsupportedCapability {
		t.Fatalf("expected unsupported capability, got %v", err)
	}
}

func TestProviderContractNormalizesSearchResult(t *testing.T) {
	provider := NewMockProvider()
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund policy")
	doc.TenantUUID = "tenant-a"
	provider.SearchResult = FixtureSearchResult("mock", doc, "refund policy")
	result, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund", TenantUUID: "tenant-a"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Provider != "mock" || result.Total != 1 || len(result.Citations) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
