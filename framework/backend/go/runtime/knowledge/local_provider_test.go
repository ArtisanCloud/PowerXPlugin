package knowledge

import (
	"context"
	"testing"
)

func TestLocalProviderSearchAndDelete(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{RequireTenant: true})
	doc := FixtureDocument("space-a", "doc-a", "Refund FAQ", "Refunds are available within 30 days.")
	doc.TenantUUID = "tenant-a"
	if _, err := provider.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	result, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund", TenantUUID: "tenant-a", SpaceIDs: []string{"space-a"}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 || result.Citations[0].DocumentID != "doc-a" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := provider.DeleteDocument(context.Background(), DeleteDocumentInput{SpaceID: "space-a", DocumentID: "doc-a"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	result, err = provider.Search(context.Background(), KnowledgeQuery{Query: "refund", TenantUUID: "tenant-a", SpaceIDs: []string{"space-a"}})
	if err != nil {
		t.Fatalf("search after delete: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected empty result after delete, got %+v", result)
	}
}

func TestLocalProviderRejectsMissingTenant(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{RequireTenant: true})
	_, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund"})
	if CodeOf(err) != CodeTenantRequired {
		t.Fatalf("expected tenant required, got %v", err)
	}
}

func TestLocalProviderListSpacesFromDocuments(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{RequireTenant: true})
	doc := FixtureDocument("space-a", "doc-a", "Refund FAQ", "Refunds are available within 30 days.")
	doc.TenantUUID = "tenant-a"
	doc.Metadata = map[string]any{"space_name": "售后知识库", "department_code": "客服"}
	if _, err := provider.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	spaces, err := provider.ListSpaces(context.Background(), ListSpacesInput{TenantUUID: "tenant-a"})
	if err != nil {
		t.Fatalf("list spaces: %v", err)
	}
	if len(spaces) != 1 || spaces[0].SpaceID != "space-a" || spaces[0].SpaceName != "售后知识库" {
		t.Fatalf("unexpected spaces: %+v", spaces)
	}
	if got := spaces[0].Metadata["document_count"]; got != 1 {
		t.Fatalf("expected document_count=1, got %#v", got)
	}
}

func TestLocalProviderListSpacesEmptyWhenNoDocuments(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{})
	spaces, err := provider.ListSpaces(context.Background(), ListSpacesInput{})
	if err != nil {
		t.Fatalf("list spaces: %v", err)
	}
	if len(spaces) != 0 {
		t.Fatalf("expected no spaces, got %+v", spaces)
	}
}

func BenchmarkLocalProviderSearchFixture(b *testing.B) {
	provider := NewLocalProvider(LocalProviderConfig{})
	for i := 0; i < 100; i++ {
		doc := FixtureDocument("space-a", "doc-"+string(rune('a'+i%26)), "Refund FAQ", "Refund policy and product help content.")
		_, _ = provider.UpsertDocument(context.Background(), doc)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Search(context.Background(), KnowledgeQuery{Query: "refund"})
	}
}
