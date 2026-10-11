package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"math"
	"strings"
	"unicode/utf8"
)

func semanticInvalid() *Error {
	return NewError(CodeInvalidArgument, "knowledge.semantic.invalid_argument")
}
func SemanticUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil
}
func validMode(value string) bool { return value == SemanticMode || value == HybridMode }
func validRole(value string) bool { return value == "source_chunk" || value == "knowledge_profile" }
func TextChecksum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func UUIDList(values []string, limit int) bool {
	if len(values) > limit {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil || seen[id.String()] {
			return false
		}
		seen[id.String()] = true
	}
	return true
}
func (in *SemanticIndexingSettings) Validate() error {
	if in == nil {
		return nil
	}
	if !validMode(in.Mode) || !SemanticUUID(in.EmbeddingProfile.UUID) || in.EmbeddingProfile.Version < 1 || len(in.ArtifactRoles) < 1 || len(in.ArtifactRoles) > 2 {
		return semanticInvalid()
	}
	seen := map[string]bool{}
	for _, role := range in.ArtifactRoles {
		if !validRole(role) || seen[role] {
			return semanticInvalid()
		}
		seen[role] = true
	}
	return nil
}
func (in SemanticQuery) Validate() error {
	if in.Schema != SemanticQuerySchema || strings.TrimSpace(in.Query) == "" || !utf8.ValidString(in.Query) || utf8.RuneCountInString(in.Query) > 8192 || !validMode(in.Mode) || in.TopK < 1 || in.TopK > 100 || len(in.SpaceUUIDs) < 1 || !UUIDList(in.SpaceUUIDs, 16) || !UUIDList(in.Filters.DocumentUUIDs, 100) || !UUIDList(in.Filters.TagUUIDs, 100) || len(in.Filters.CategoryCodes) > 100 || len(in.Filters.ArtifactRoles) > 100 {
		return semanticInvalid()
	}
	for _, role := range in.Filters.ArtifactRoles {
		if !validRole(role) {
			return semanticInvalid()
		}
	}
	for _, code := range in.Filters.CategoryCodes {
		if strings.TrimSpace(code) == "" || len(code) > 128 {
			return semanticInvalid()
		}
	}
	for space, gen := range in.RequiredGenerations {
		if !containsUUID(in.SpaceUUIDs, space) || !SemanticUUID(gen) {
			return semanticInvalid()
		}
	}
	return nil
}
func containsUUID(values []string, target string) bool {
	id, e := uuid.Parse(target)
	if e != nil {
		return false
	}
	for _, value := range values {
		parsed, e := uuid.Parse(value)
		if e == nil && parsed == id {
			return true
		}
	}
	return false
}
func (in KnowledgeDocument) ValidateSemantic() error {
	if len(in.IdempotencyKey) > 128 || strings.TrimSpace(in.IdempotencyKey) != in.IdempotencyKey {
		return semanticInvalid()
	}
	if in.Indexing == nil {
		if len(in.Artifacts) > 0 || in.ExternalRef != nil {
			return semanticInvalid()
		}
		return nil
	}
	if err := in.Indexing.Validate(); err != nil {
		return err
	}
	if len(in.Artifacts) < 1 || len(in.Artifacts) > 64 || in.ExternalRef != nil && !SemanticUUID(in.ExternalRef.DocumentUUID) {
		return semanticInvalid()
	}
	seen, found := map[string]bool{}, map[string]bool{}
	total := 0
	for _, a := range in.Artifacts {
		if !SemanticUUID(a.UUID) || seen[a.UUID] || !containsString(in.Indexing.ArtifactRoles, a.Role) || strings.TrimSpace(a.Text) == "" || !utf8.ValidString(a.Text) || a.Version == "" || a.Checksum != TextChecksum(a.Text) {
			return semanticInvalid()
		}
		seen[a.UUID] = true
		found[a.Role] = true
		total += len(a.Text)
		if total > 8<<20 || len(a.CategoryCodes) > 32 || !UUIDList(a.TagUUIDs, 64) {
			return semanticInvalid()
		}
		for _, code := range a.CategoryCodes {
			if code == "" || code != strings.TrimSpace(code) || len(code) > 128 {
				return semanticInvalid()
			}
		}
		for _, id := range []*string{a.KnowledgeProfileUUID, a.CaseUUID} {
			if id != nil && !SemanticUUID(*id) {
				return semanticInvalid()
			}
		}
		r := a.SourceRef
		if !SemanticUUID(r.SourceUUID) || r.Version != a.Version || r.Checksum != a.Checksum || r.PositionUnit != "unicode_codepoint" || r.CharStart != 0 || r.CharEnd != utf8.RuneCountInString(a.Text) {
			return semanticInvalid()
		}
		if a.Role == "knowledge_profile" && (r.Kind != "knowledge_profile" || a.KnowledgeProfileUUID == nil || *a.KnowledgeProfileUUID != r.SourceUUID) {
			return semanticInvalid()
		}
		if a.Role == "source_chunk" && r.Kind != "raw_document" && r.Kind != "cleaned_document" {
			return semanticInvalid()
		}
	}
	for _, role := range in.Indexing.ArtifactRoles {
		if !found[role] {
			return semanticInvalid()
		}
	}
	return nil
}
func containsString(values []string, value string) bool {
	for _, s := range values {
		if s == value {
			return true
		}
	}
	return false
}

// SourceText uses codepoint offsets on the immutable full source, never byte offsets.
func SourceText(source string, ref SemanticSourceRef) (string, error) {
	if !utf8.ValidString(source) || ref.PositionUnit != "unicode_codepoint" || ref.Checksum != TextChecksum(source) {
		return "", NewError(CodeInvalidResponse, "knowledge.semantic.invalid_source")
	}
	chars := []rune(source)
	if ref.CharStart < 0 || ref.CharEnd < ref.CharStart || ref.CharEnd > len(chars) {
		return "", NewError(CodeInvalidResponse, "knowledge.semantic.invalid_source")
	}
	return string(chars[ref.CharStart:ref.CharEnd]), nil
}
func (in SemanticResult) Validate(query SemanticQuery) error {
	bad := func() error { return NewError(CodeInvalidResponse, "knowledge.semantic.invalid_response") }
	if in.Schema != SemanticResultSchema || !SemanticUUID(in.QueryUUID) || in.RequestedMode != query.Mode || in.EffectiveMode != query.Mode || in.Items == nil || len(in.Items) > query.TopK || len(in.Generations) != len(query.SpaceUUIDs) {
		return bad()
	}
	generations := map[string]string{}
	for _, g := range in.Generations {
		if !containsUUID(query.SpaceUUIDs, g.SpaceUUID) || generations[g.SpaceUUID] != "" || !SemanticUUID(g.ConfigurationGeneration) || !SemanticUUID(g.CorpusGeneration) || !SemanticUUID(g.EmbeddingProfile.UUID) || g.EmbeddingProfile.Version < 1 || g.Dimensions < 1 || g.ModelKey == "" || g.ModelRevision == "" || g.DistanceMetric != "cosine" {
			return bad()
		}
		generations[g.SpaceUUID] = g.CorpusGeneration
	}
	for space, gen := range query.RequiredGenerations {
		if generations[space] != gen {
			return bad()
		}
	}
	for _, item := range in.Items {
		if !containsUUID(query.SpaceUUIDs, item.SpaceUUID) || !SemanticUUID(item.DocumentUUID) || !SemanticUUID(item.ChunkUUID) || !SemanticUUID(item.ArtifactUUID) || !validRole(item.ArtifactRole) || item.DocumentVersion == "" || !SemanticUUID(item.BundleGeneration) || item.Title == "" || !utf8.ValidString(item.Text) || item.Text == "" || math.IsNaN(item.Score) || math.IsInf(item.Score, 0) || item.SourceRef.PositionUnit != "unicode_codepoint" || item.SourceRef.CharStart < 0 || item.SourceRef.CharEnd-item.SourceRef.CharStart != utf8.RuneCountInString(item.Text) || !SemanticUUID(item.SourceRef.SourceUUID) || item.SourceRef.Version == "" || len(item.SourceRef.Checksum) != 64 || len(item.Scores) == 0 || len(item.RetrievalSources) == 0 {
			return bad()
		}
		if query.Mode == SemanticMode && item.ScoreType != "cosine_similarity" || query.Mode == HybridMode && item.ScoreType != "rrf" {
			return bad()
		}
		for _, score := range item.Scores {
			if math.IsNaN(score.Score) || math.IsInf(score.Score, 0) || score.Source == "" || (score.ScoreType == "cosine_distance" && score.Direction != "lower_better") || (score.ScoreType != "cosine_distance" && score.Direction != "higher_better") || !containsString([]string{"cosine_distance", "cosine_similarity", "postgres_text_rank", "rrf"}, score.ScoreType) {
				return bad()
			}
		}
	}
	return nil
}
