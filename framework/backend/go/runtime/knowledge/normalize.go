package knowledge

import (
	"strings"
	"time"
)

func NormalizeSearchResult(provider KnowledgeProvider, query KnowledgeQuery, result *KnowledgeSearchResult) (*KnowledgeSearchResult, error) {
	if result == nil {
		result = &KnowledgeSearchResult{}
	}
	query = query.Normalized()
	result.Provider = strings.TrimSpace(result.Provider)
	if result.Provider == "" && provider != nil {
		result.Provider = provider.Name()
	}
	result.TraceID = strings.TrimSpace(result.TraceID)
	if result.TraceID == "" {
		result.TraceID = query.TraceID
	}
	result.Citations = result.Citations[:0]
	normalizedChunks := make([]KnowledgeChunk, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		chunk.Text = strings.TrimSpace(chunk.Text)
		chunk.DocumentID = strings.TrimSpace(chunk.DocumentID)
		chunk.SpaceID = strings.TrimSpace(chunk.SpaceID)
		chunk.TenantUUID = strings.TrimSpace(chunk.TenantUUID)
		if chunk.Score < 0 {
			chunk.Score = 0
		}
		if chunk.Score > 1 {
			chunk.Score = 1
		}
		if chunk.TenantUUID != "" && query.TenantUUID != "" && chunk.TenantUUID != query.TenantUUID {
			return nil, NewError(CodeTenantMismatch, "knowledge result tenant mismatch")
		}
		if chunk.Citation == nil {
			return nil, NewError(CodeInvalidDocument, ErrCitationRequired.Error())
		}
		chunk.Citation.DocumentID = strings.TrimSpace(chunk.Citation.DocumentID)
		chunk.Citation.Title = strings.TrimSpace(chunk.Citation.Title)
		if chunk.Citation.Provider == "" {
			chunk.Citation.Provider = result.Provider
		}
		if chunk.Citation.RetrievedAt.IsZero() {
			chunk.Citation.RetrievedAt = time.Now().UTC()
		}
		if err := chunk.Validate(); err != nil {
			return nil, err
		}
		chunk.Metadata = RedactMap(chunk.Metadata)
		if chunk.Citation.Position != nil {
			chunk.Citation.Position = KnowledgePosition(RedactMap(map[string]any(chunk.Citation.Position)))
		}
		normalizedChunks = append(normalizedChunks, chunk)
		result.Citations = append(result.Citations, *chunk.Citation)
	}
	result.Chunks = normalizedChunks
	result.Total = len(result.Chunks)
	result.Diagnostics = RedactMap(result.Diagnostics)
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	return result, nil
}
