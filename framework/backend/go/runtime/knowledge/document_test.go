package knowledge

import (
	"context"
	"testing"
)

func TestDocumentReindexJob(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{})
	doc := FixtureDocument("space-a", "doc-a", "FAQ", "refund")
	if _, err := provider.UpsertDocument(context.Background(), doc); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	job, err := provider.Reindex(context.Background(), ReindexInput{SpaceID: "space-a", DocumentID: "doc-a"})
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if job.Status != IndexStatusSucceeded || job.Operation != IndexOperationReindex {
		t.Fatalf("unexpected job: %+v", job)
	}
}
