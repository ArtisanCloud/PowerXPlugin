package knowledge

import "testing"

func TestCompletedIndexJob(t *testing.T) {
	doc := KnowledgeDocument{SpaceID: "space-a", DocumentID: "doc-a", Version: "v1"}
	job := CompletedIndexJob(IndexOperationUpsert, doc)
	if job.Status != IndexStatusSucceeded || job.JobID == "" || job.FinishedAt.IsZero() {
		t.Fatalf("unexpected job: %+v", job)
	}
}
