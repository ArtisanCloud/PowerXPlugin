package knowledge

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

func TestLocalProviderAssignsUUIDAndTracksSpaceRebuildJob(t *testing.T) {
	provider := NewLocalProvider(LocalProviderConfig{})
	job, err := provider.UpsertDocument(context.Background(), KnowledgeDocument{SpaceID: "space-a", Title: "FAQ", URI: "local://faq", Content: "refund"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := uuid.Parse(job.DocumentID); err != nil {
		t.Fatalf("document UUID = %q: %v", job.DocumentID, err)
	}
	rebuild, err := provider.Reindex(context.Background(), ReindexInput{SpaceID: "space-a"})
	if err != nil || rebuild.DocumentID != "" || rebuild.Status != IndexStatusSucceeded {
		t.Fatalf("space rebuild = %#v, %v", rebuild, err)
	}
	stored, err := provider.GetIndexJob(context.Background(), IndexJobQuery{JobID: rebuild.JobID})
	if err != nil || stored.JobID != rebuild.JobID {
		t.Fatalf("GetIndexJob() = %#v, %v", stored, err)
	}
}
