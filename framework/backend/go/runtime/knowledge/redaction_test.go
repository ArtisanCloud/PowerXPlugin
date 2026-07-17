package knowledge

import "testing"

func TestNormalizeSearchResultRedactsMetadata(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund")
	result := FixtureSearchResult("mock", doc, "refund")
	result.Chunks[0].Metadata = map[string]any{"secret": "abc", "safe": "ok"}
	normalized, err := NormalizeSearchResult(NewMockProvider(), KnowledgeQuery{Query: "refund"}, result)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.Chunks[0].Metadata["secret"] != "[redacted]" {
		t.Fatalf("expected redacted metadata: %+v", normalized.Chunks[0].Metadata)
	}
}
