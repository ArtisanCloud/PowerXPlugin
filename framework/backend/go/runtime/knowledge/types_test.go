package knowledge

import "testing"

func TestKnowledgeQueryNormalizeAndTenantRequired(t *testing.T) {
	q := KnowledgeQuery{Query: "  refund  ", Limit: 999, MinScore: 2, Visibility: VisibilityTenant}
	q = q.Normalized()
	if q.Query != "refund" || q.Limit != MaxQueryLimit || q.MinScore != 1 {
		t.Fatalf("unexpected normalized query: %+v", q)
	}
	if err := q.Validate(true); CodeOf(err) != CodeTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
}

func TestKnowledgeDocumentValidation(t *testing.T) {
	_, err := ValidateDocument(KnowledgeDocument{DocumentID: "doc-1", SpaceID: "space-1", Title: "Doc"})
	if CodeOf(err) != CodeInvalidDocument {
		t.Fatalf("expected invalid document, got %v", err)
	}
	doc, err := ValidateDocument(KnowledgeDocument{DocumentID: "doc-1", SpaceID: "space-1", Title: "Doc", Content: "hello"})
	if err != nil {
		t.Fatalf("validate document: %v", err)
	}
	if doc.Checksum == "" || doc.Version == "" {
		t.Fatalf("expected checksum/version to be filled: %+v", doc)
	}
}

func TestChunkRequiresCitation(t *testing.T) {
	err := (KnowledgeChunk{Text: "hello"}).Validate()
	if CodeOf(err) != CodeInvalidDocument {
		t.Fatalf("expected invalid document for missing citation, got %v", err)
	}
}
