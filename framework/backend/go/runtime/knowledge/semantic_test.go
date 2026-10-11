package knowledge

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSemanticSourceUsesCodepointsAndOpaqueVersions(t *testing.T) {
	text := "中文😀é"
	ref := SemanticSourceRef{PositionUnit: "unicode_codepoint", Checksum: TextChecksum(text), CharStart: 2, CharEnd: 5}
	value, err := SourceText(text, ref)
	require.NoError(t, err)
	require.Equal(t, "😀é", value)
	artifact := SemanticArtifact{UUID: "11111111-1111-4111-8111-111111111111", Role: "source_chunk", Text: text, Checksum: TextChecksum(text), Version: "opaque/source-v1", CategoryCodes: []string{}, TagUUIDs: []string{}, SourceRef: SemanticSourceRef{Kind: "raw_document", SourceUUID: "22222222-2222-4222-8222-222222222222", Version: "opaque/source-v1", Checksum: TextChecksum(text), PositionUnit: "unicode_codepoint", CharEnd: 5}}
	document := KnowledgeDocument{Indexing: &SemanticIndexingSettings{Mode: HybridMode, EmbeddingProfile: DocumentProfileRef{UUID: "33333333-3333-4333-8333-333333333333", Version: 7}, ArtifactRoles: []string{"source_chunk"}}, Artifacts: []SemanticArtifact{artifact}}
	require.NoError(t, document.ValidateSemantic())
	raw, err := json.Marshal(artifact)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"tag_uuids":[]`)
	document.Artifacts[0].SourceRef.CharEnd = len(text)
	require.Error(t, document.ValidateSemantic())
}
func TestLocalNeverUsesLexicalForSemanticRequests(t *testing.T) {
	local := NewLocalProvider(LocalProviderConfig{})
	_, err := local.QuerySemanticKnowledge(context.Background(), SemanticQuery{})
	require.Equal(t, CodeUnsupportedCapability, CodeOf(err))
	_, err = local.UpsertDocument(context.Background(), KnowledgeDocument{Indexing: &SemanticIndexingSettings{Mode: SemanticMode}})
	require.Equal(t, CodeUnsupportedCapability, CodeOf(err))
}
func TestSemanticQueryIsNotNormalizedOrScoredWithLegacyDefaults(t *testing.T) {
	text := "  需要处理声誉问题😀，保留完整句子。  "
	query := SemanticQuery{Schema: SemanticQuerySchema, Query: text, SpaceUUIDs: []string{"11111111-1111-4111-8111-111111111111"}, Mode: SemanticMode, TopK: 20, Filters: SemanticFilters{DocumentUUIDs: []string{}, CategoryCodes: []string{"public_relations"}, TagUUIDs: []string{}, ArtifactRoles: []string{"knowledge_profile", "source_chunk"}}}
	require.NoError(t, query.Validate())
	raw, err := json.Marshal(query)
	require.NoError(t, err)
	require.Contains(t, string(raw), text)
	require.NotContains(t, string(raw), "min_score")
	require.Equal(t, []string{"knowledge_profile", "source_chunk"}, query.Filters.ArtifactRoles)
	query.Query = strings.Repeat("😀", 8193)
	require.Error(t, query.Validate())
}
