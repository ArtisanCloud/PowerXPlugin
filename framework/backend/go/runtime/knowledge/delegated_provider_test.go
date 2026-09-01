package knowledge

import (
	"context"
	"testing"
	"time"
)

type fakeDelegatedClient struct {
	result *KnowledgeSearchResult
	spaces []KnowledgeSpace
	err    error
	delay  time.Duration
}

func (fakeDelegatedClient) DelegatedCapabilities(context.Context) ProviderCapabilities {
	return BasicCapabilities("fake", ProviderModeDelegated, OperationRetrieve, OperationSearch, OperationUpsert, OperationDelete, OperationReindex, OperationHealth)
}

func (f fakeDelegatedClient) ListKnowledgeSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.spaces, f.err
}

func (f fakeDelegatedClient) SearchKnowledge(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.result, f.err
}

func (f fakeDelegatedClient) UpsertKnowledgeDocument(context.Context, KnowledgeDocument) (*KnowledgeIndexJob, error) {
	return nil, f.err
}

func (f fakeDelegatedClient) DeleteKnowledgeDocument(context.Context, DeleteDocumentInput) (*KnowledgeIndexJob, error) {
	return nil, f.err
}

func (f fakeDelegatedClient) ReindexKnowledgeDocument(context.Context, ReindexInput) (*KnowledgeIndexJob, error) {
	return nil, f.err
}

func TestDelegatedProviderNormalizesSuccess(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund policy")
	provider := NewDelegatedProvider(DelegatedProviderConfig{Client: fakeDelegatedClient{result: FixtureSearchResult("powerx_delegated", doc, "refund policy")}})
	result, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Provider != "powerx_delegated" || result.Total != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDelegatedProviderListSpaces(t *testing.T) {
	provider := NewDelegatedProvider(DelegatedProviderConfig{Client: fakeDelegatedClient{spaces: []KnowledgeSpace{{SpaceID: "space-a", SpaceName: "售后知识库"}}}})
	spaces, err := provider.ListSpaces(context.Background(), ListSpacesInput{TenantUUID: "tenant-a"})
	if err != nil {
		t.Fatalf("list spaces: %v", err)
	}
	if len(spaces) != 1 || spaces[0].SpaceID != "space-a" {
		t.Fatalf("unexpected spaces: %+v", spaces)
	}
}

func TestDelegatedProviderRejectsTenantMismatchResult(t *testing.T) {
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund policy")
	doc.TenantUUID = "tenant-b"
	provider := NewDelegatedProvider(DelegatedProviderConfig{Client: fakeDelegatedClient{result: FixtureSearchResult("powerx_delegated", doc, "refund policy")}})
	_, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund", TenantUUID: "tenant-a", Visibility: VisibilityTenant})
	if CodeOf(err) != CodeTenantMismatch {
		t.Fatalf("expected tenant mismatch, got %v", err)
	}
}

func TestDelegatedProviderTimeout(t *testing.T) {
	provider := NewDelegatedProvider(DelegatedProviderConfig{Client: fakeDelegatedClient{delay: 50 * time.Millisecond}, Timeout: time.Millisecond})
	_, err := provider.Search(context.Background(), KnowledgeQuery{Query: "refund"})
	if CodeOf(err) != CodeProviderUnavailable {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
}
