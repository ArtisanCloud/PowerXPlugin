package knowledge

import (
	"context"
	"encoding/json"
	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestHostRebuildUsesExactScopeAndExplicitMode(t *testing.T) {
	const space = "11111111-1111-4111-8111-111111111111"
	const doc = "22222222-2222-4222-8222-222222222222"
	const job = "33333333-3333-4333-8333-333333333333"
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		for _, document := range []string{"", doc} {
			for _, mode := range []fwknowledge.DocumentRebuildMode{fwknowledge.RebuildReuseSnapshot, fwknowledge.RebuildApplyConfig} {
				t.Run(scheme+"/"+document+"/"+string(mode), func(t *testing.T) {
					settings := &fwknowledge.DocumentRebuildSettings{Mode: mode, IdempotencyKey: "fixture-key"}
					if mode == fwknowledge.RebuildApplyConfig {
						v := false
						zero := 0
						empty := []string{}
						settings.Ingestion = &fwknowledge.DocumentIngestionSettings{Schema: fwknowledge.DocumentIngestionSchema, PagePriority: &v, ChunkOverlap: &zero, Separators: &empty}
					}
					calls := 0
					transport := &http.Client{Transport: documentTransportFixture(t, scheme, func(r *http.Request) (*http.Response, error) {
						calls++
						require.Equal(t, scheme+" fixture", r.Header.Get("Authorization"))
						require.Equal(t, http.MethodPost, r.Method)
						want := "/api/v1/tenant/knowledge/spaces/" + space
						if document != "" {
							want += "/documents/" + document
						}
						require.Equal(t, want+"/indexes:rebuild", r.URL.Path)
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, string(mode), body["mode"])
						require.Equal(t, "fixture-key", body["idempotency_key"])
						require.NotContains(t, body, "tenant_uuid")
						require.NotContains(t, body, "document_uuid")
						if mode == fwknowledge.RebuildReuseSnapshot {
							require.NotContains(t, body, "ingestion")
						} else {
							config := body["ingestion"].(map[string]any)
							require.Equal(t, false, config["page_priority"])
							require.Equal(t, float64(0), config["chunk_overlap"])
							require.Equal(t, []any{}, config["separators"])
						}
						item := map[string]any{"job_uuid": job, "operation": "rebuild", "status": "queued"}
						if document != "" {
							item["document_uuid"] = document
						}
						raw, _ := json.Marshal(map[string]any{"data": item})
						return knowledgeResponse(202, string(raw)), nil
					})}
					var client *Client
					var err error
					if scheme == "ApiKey" {
						client, err = NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "fixture", transport)
					} else {
						client, err = NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) { return "fixture", nil }), transport)
					}
					require.NoError(t, err)
					for n := 0; n < 2; n++ {
						result, err := client.ReindexKnowledgeDocument(context.Background(), fwknowledge.ReindexInput{SpaceID: space, DocumentID: document, HostRebuild: settings})
						require.NoError(t, err)
						require.Equal(t, job, result.JobID)
						require.Equal(t, document, result.DocumentID)
					}
					require.Equal(t, 2, calls)
				})
			}
		}
	}
}

func TestHostRebuildRejectsInvalidInputAndDoesNotRetryBroaderScope(t *testing.T) {
	calls := 0
	client, err := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "fixture", &http.Client{Transport: knowledgeRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		require.True(t, strings.Contains(r.URL.Path, "/documents/"))
		return knowledgeResponse(409, `{"reason_code":"KNOWLEDGE_INDEX_CONFLICT"}`), nil
	})})
	require.NoError(t, err)
	req := fwknowledge.ReindexInput{SpaceID: "11111111-1111-4111-8111-111111111111", DocumentID: "22222222-2222-4222-8222-222222222222", HostRebuild: &fwknowledge.DocumentRebuildSettings{Mode: fwknowledge.RebuildApplyConfig, IdempotencyKey: "fixture"}}
	_, err = client.ReindexKnowledgeDocument(context.Background(), req)
	require.Equal(t, fwknowledge.CodeInvalidArgument, fwknowledge.CodeOf(err))
	require.Zero(t, calls)
	req.HostRebuild.Mode = fwknowledge.RebuildReuseSnapshot
	_, err = client.ReindexKnowledgeDocument(context.Background(), req)
	require.Equal(t, 409, fwknowledge.HTTPStatus(err))
	require.Equal(t, 1, calls)
}
