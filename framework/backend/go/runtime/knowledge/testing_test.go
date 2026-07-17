package knowledge

import (
	"context"
	"testing"
)

func TestMockProviderOutcomes(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund")
	success := NewMockProvider()
	success.SearchResult = FixtureSearchResult("mock", doc, "refund")
	if result, err := success.Search(context.Background(), KnowledgeQuery{Query: "refund"}); err != nil || result.Total != 1 {
		t.Fatalf("success outcome failed: result=%+v err=%v", result, err)
	}

	empty := NewMockProvider()
	if result, err := empty.Search(context.Background(), KnowledgeQuery{Query: "missing"}); err != nil || result.Total != 0 {
		t.Fatalf("empty outcome failed: result=%+v err=%v", result, err)
	}

	unsupported := NewMockProvider()
	unsupported.SearchErr = Unsupported(OperationSearch)
	if _, err := unsupported.Search(context.Background(), KnowledgeQuery{Query: "refund"}); CodeOf(err) != CodeUnsupportedCapability {
		t.Fatalf("unsupported outcome failed: %v", err)
	}

	denied := NewMockProvider()
	denied.SearchErr = NewError(CodeForbidden, "access denied")
	if _, err := denied.Search(context.Background(), KnowledgeQuery{Query: "refund"}); CodeOf(err) != CodeForbidden {
		t.Fatalf("access denied outcome failed: %v", err)
	}

	unavailable := NewMockProvider()
	unavailable.SearchErr = NewError(CodeProviderUnavailable, "unavailable")
	if _, err := unavailable.Search(context.Background(), KnowledgeQuery{Query: "refund"}); CodeOf(err) != CodeProviderUnavailable {
		t.Fatalf("unavailable outcome failed: %v", err)
	}
}
