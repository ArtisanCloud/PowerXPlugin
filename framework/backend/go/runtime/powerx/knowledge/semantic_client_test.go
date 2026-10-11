package knowledge

import (
	"context"
	"encoding/json"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

const semSpace = "11111111-1111-4111-8111-111111111111"
const semDoc = "22222222-2222-4222-8222-222222222222"
const semProfile = "33333333-3333-4333-8333-333333333333"

func semanticGenerationFixture() fw.SemanticGeneration {
	return fw.SemanticGeneration{Env: "dev", SpaceUUID: semSpace, ConfigurationGeneration: semProfile, CorpusGeneration: semDoc, EmbeddingProfile: fw.DocumentProfileRef{UUID: semProfile, Version: 7}, ModelKey: "fixture-model", ModelRevision: strings.Repeat("a", 64), Dimensions: 3, DistanceMetric: "cosine"}
}
func TestSemanticTransportAndScoresPreserveAuthority(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			generation := semanticGenerationFixture()
			queryText := "  品牌遇到质疑😀，希望借鉴修复信誉的案例。  "
			transport := &http.Client{Transport: knowledgeRoundTrip(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, scheme+" fixture-token", r.Header.Get("Authorization"))
				require.Empty(t, r.Header.Get("X-Tenant-UUID"))
				var body map[string]any
				if r.Body != nil {
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				}
				operation := ""
				if scheme == "Bearer" {
					require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
					require.Equal(t, "core_internal", body["preferred_protocol"])
					payload := body["payload"].(map[string]any)
					require.Equal(t, "INVOKE", payload["method"])
					require.Contains(t, []string{"core://knowledge/retrieval", "core://knowledge/documents"}, payload["endpoint"])
					body = payload["body"].(map[string]any)
					operation = body["operation"].(string)
				} else {
					switch {
					case strings.HasSuffix(r.URL.Path, "/semantic-index") && r.Method == "GET":
						operation = "capabilities"
					case strings.HasSuffix(r.URL.Path, "/semantic-index"):
						operation = "configure_semantic_index"
					case strings.HasSuffix(r.URL.Path, "/retrieval/query"):
						operation = "query"
					case strings.HasSuffix(r.URL.Path, "/visibility"):
						operation = "set_document_visibility"
					}
				}
				var business any
				switch operation {
				case "capabilities":
					item := fw.SemanticCapabilities{Schema: fw.SemanticCapabilitiesSchema, Configured: true, Ready: true, Indexed: true, Generation: &generation, QueryModes: []string{"semantic", "hybrid"}, PendingGeneration: nil}
					if scheme == "Bearer" {
						business = map[string]any{"capabilities": item}
					} else {
						business = item
					}
				case "configure_semantic_index":
					configuration := body
					if scheme == "Bearer" {
						configuration = body["configuration"].(map[string]any)
					}
					require.Equal(t, "fixture-model", configuration["embedding_profile_key"])
					if scheme == "Bearer" {
						business = map[string]any{"item": generation}
					} else {
						business = generation
					}
				case "query":
					q := body
					if scheme == "Bearer" {
						q = body["query"].(map[string]any)
					}
					require.Equal(t, queryText, q["query"])
					require.NotContains(t, q, "min_score")
					require.NotContains(t, q, "model")
					match := fw.SemanticMatch{SpaceUUID: semSpace, DocumentUUID: semDoc, ChunkUUID: semProfile, ArtifactUUID: semProfile, ArtifactRole: "source_chunk", Title: "fixture", Text: "中😀文", Score: -0.2, ScoreType: "cosine_similarity", Scores: []fw.SemanticScore{{Source: "vector", Score: 1.2, ScoreType: "cosine_distance", Direction: "lower_better"}, {Source: "vector", Score: -0.2, ScoreType: "cosine_similarity", Direction: "higher_better"}}, RetrievalSources: []string{"vector"}, SourceRef: fw.SemanticSourceRef{Kind: "raw_document", SourceUUID: semDoc, Checksum: fw.TextChecksum("中😀文"), Version: "source/v1", PositionUnit: "unicode_codepoint", CharEnd: 3}, DocumentVersion: "source/v1", BundleGeneration: semDoc}
					result := fw.SemanticResult{Schema: fw.SemanticResultSchema, QueryUUID: semDoc, RequestedMode: "semantic", EffectiveMode: "semantic", Generations: []fw.SemanticGeneration{generation}, Items: []fw.SemanticMatch{match, match}, TraceID: "business-trace"}
					if scheme == "Bearer" {
						business = map[string]any{"result": result}
					} else {
						business = result
					}
				case "set_document_visibility":
					visibility := body
					if scheme == "Bearer" {
						visibility = body["visibility"].(map[string]any)
					}
					require.Equal(t, false, visibility["queryable"])
					business = fw.SemanticVisibility{DocumentUUID: semDoc, Queryable: false, VisibilityEpoch: semProfile}
				default:
					t.Fatalf("unknown operation %s", operation)
				}
				if scheme == "Bearer" {
					business = map[string]any{"payload": business, "trace_id": "invocation-trace"}
				}
				raw, _ := json.Marshal(map[string]any{"data": business, "request_id": "request-fixture"})
				return knowledgeResponse(200, string(raw)), nil
			})}
			var client *Client
			var err error
			if scheme == "ApiKey" {
				client, err = NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "fixture-token", transport)
			} else {
				client, err = NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) { return "fixture-token", nil }), transport)
			}
			require.NoError(t, err)
			caps, err := client.GetSemanticCapabilities(context.Background(), semSpace)
			require.NoError(t, err)
			require.True(t, caps.Indexed)
			_, err = client.ConfigureSemanticIndex(context.Background(), semSpace, fw.SemanticConfigureInput{EmbeddingProfileKey: "fixture-model"})
			require.NoError(t, err)
			result, err := client.QuerySemanticKnowledge(context.Background(), fw.SemanticQuery{Schema: fw.SemanticQuerySchema, Query: queryText, SpaceUUIDs: []string{semSpace}, Mode: "semantic", TopK: 3, Filters: fw.SemanticFilters{DocumentUUIDs: []string{}, TagUUIDs: []string{}, CategoryCodes: []string{}, ArtifactRoles: []string{}}})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			require.Equal(t, -0.2, result.Items[0].Score)
			require.Equal(t, 1.2, result.Items[0].Scores[0].Score)
			require.Equal(t, "business-trace", result.TraceID)
			require.Equal(t, "request-fixture", result.RequestID)
			if scheme == "Bearer" {
				require.Equal(t, "invocation-trace", result.InvocationTraceID)
			}
			value := false
			_, err = client.SetDocumentVisibility(context.Background(), semSpace, semDoc, fw.SemanticVisibilityInput{Queryable: &value})
			require.NoError(t, err)
		})
	}
}
func TestSemanticErrorsNeverFallBackOrBecomeEmptySuccess(t *testing.T) {
	for _, status := range []int{403, 409, 422, 501, 503} {
		calls := 0
		client, err := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "fixture", &http.Client{Transport: knowledgeRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			require.Equal(t, "/api/v1/tenant/knowledge/retrieval/query", r.URL.Path)
			return knowledgeResponse(status, `{"reason_code":"KNOWLEDGE_CHANNEL_FAILURE","trace_id":"core-trace","request_id":"core-request","details":{"controlled":"fixture"}}`), nil
		})})
		require.NoError(t, err)
		result, err := client.QuerySemanticKnowledge(context.Background(), fw.SemanticQuery{Schema: fw.SemanticQuerySchema, Query: "fixture", SpaceUUIDs: []string{semSpace}, Mode: "hybrid", TopK: 1})
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, status, fw.HTTPStatus(err))
		require.Equal(t, 1, calls)
		var typed *fw.Error
		require.ErrorAs(t, err, &typed)
		require.Equal(t, "core-trace", typed.TraceID)
		require.Equal(t, "core-request", typed.RequestID)
	}
}
