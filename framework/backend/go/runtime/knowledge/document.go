package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

func ValidateDocument(document KnowledgeDocument) (KnowledgeDocument, error) {
	document = document.Normalized()
	if err := document.Validate(); err != nil {
		return KnowledgeDocument{}, err
	}
	if document.Checksum == "" && document.Content != "" {
		sum := sha256.Sum256([]byte(document.Content))
		document.Checksum = hex.EncodeToString(sum[:])
	}
	if document.Version == "" {
		document.Version = document.Checksum
	}
	document.Metadata = RedactMap(document.Metadata)
	return document, nil
}

func CompletedIndexJob(operation string, document KnowledgeDocument) *KnowledgeIndexJob {
	now := time.Now().UTC()
	return &KnowledgeIndexJob{
		JobID:      uuid.NewString(),
		SpaceID:    document.SpaceID,
		DocumentID: document.DocumentID,
		Operation:  operation,
		Status:     IndexStatusSucceeded,
		StartedAt:  now,
		FinishedAt: now,
	}
}
