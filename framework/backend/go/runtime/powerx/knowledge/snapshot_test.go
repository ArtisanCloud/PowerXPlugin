package knowledge

import (
	"context"
	"encoding/json"
	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestHostSnapshotPreservesExplicitSettingsAndReadback(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			const jobID = "33333333-3333-4333-8333-333333333333"
			const spaceID = "11111111-1111-4111-8111-111111111111"
			var settings fwknowledge.DocumentIngestionSettings
			require.NoError(t, json.Unmarshal([]byte(`{"schema":"powerx.knowledge.document-ingestion/v1","ingestion_profile":{"uuid":"22222222-2222-4222-8222-222222222222","version":7},"priority":"high","chunk_size":0,"chunk_overlap":0,"page_priority":false,"separators":[],"segment_order":["page","segment","separator","size"]}`), &settings))
			transport := &http.Client{Transport: documentTransportFixture(t, scheme, func(r *http.Request) (*http.Response, error) {
				require.Equal(t, scheme+" service-token", r.Header.Get("Authorization"))
				if r.Method == http.MethodPost {
					require.Equal(t, "/api/v1/tenant/knowledge/spaces/"+spaceID+"/documents", r.URL.Path)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.NotContains(t, body, "tenant_uuid")
					require.NotContains(t, body, "segmentMode")
					require.Equal(t, "\n fixture \n", body["content"])
					snapshot := body["ingestion"].(map[string]any)
					require.Equal(t, float64(0), snapshot["chunk_size"])
					require.Equal(t, false, snapshot["page_priority"])
					require.Equal(t, []any{}, snapshot["separators"])
					require.Equal(t, []any{"page", "segment", "separator", "size"}, snapshot["segment_order"])
					require.NotContains(t, snapshot, "processor_profile")
					return knowledgeResponse(202, `{"data":{"job_uuid":"`+jobID+`","document_uuid":"22222222-2222-4222-8222-222222222222","status":"queued"}}`), nil
				}
				if r.URL.Path == "/api/v1/tenant/knowledge/index-jobs/"+jobID+"/chunks" {
					return knowledgeResponse(200, `{"data":{"items":[{"uuid":"44444444-4444-4444-8444-444444444444","job_uuid":"`+jobID+`","space_uuid":"`+spaceID+`","document_uuid":"22222222-2222-4222-8222-222222222222","ordinal":0,"content":"fixture","checksum":"fixture","metadata":{"snapshot_checksum":"fixture"}}]}}`), nil
				}
				require.Equal(t, "/api/v1/tenant/knowledge/index-jobs/"+jobID, r.URL.Path)
				return knowledgeResponse(200, `{"data":{"job_uuid":"`+jobID+`","status":"succeeded","priority":"high","trace_id":"fixture","chunk_count":4,"snapshot_checksum":"fixture","effective_config":{"schema":"powerx.knowledge.document-ingestion/v1","index_mode":"lexical_chunks","length_unit":"unicode_rune","chunk_size":640,"chunk_overlap":80},"requested_config":{"schema":"powerx.knowledge.document-ingestion/v1","chunk_size":640}}}`), nil
			})}
			var client *Client
			var err error
			if scheme == "ApiKey" {
				client, err = NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "service-token", transport)
			} else {
				client, err = NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) { return "service-token", nil }), transport)
			}
			require.NoError(t, err)
			_, err = client.UpsertKnowledgeDocument(context.Background(), fwknowledge.KnowledgeDocument{SpaceID: spaceID, Title: "fixture", URI: "plugin://fixture", Content: "\n fixture \n", ContentType: "text/plain", Version: "v1", HostIngestion: &settings})
			require.NoError(t, err)
			job, err := client.GetKnowledgeIndexJob(context.Background(), fwknowledge.IndexJobQuery{JobID: jobID})
			require.NoError(t, err)
			require.Equal(t, 4, *job.ChunkCount)
			require.Equal(t, 640, job.EffectiveConfig.ChunkSize)
			require.Equal(t, "fixture", job.TraceID)
			require.Equal(t, 640, *job.RequestedConfig.ChunkSize)
			chunks, err := client.GetKnowledgeIndexJobChunks(context.Background(), fwknowledge.IndexJobQuery{JobID: jobID})
			require.NoError(t, err)
			require.Len(t, chunks, 1)
			require.Equal(t, jobID, chunks[0].JobUUID)
		})
	}
}

func TestHostSnapshotPreservesCoreRejections(t *testing.T) {
	for _, tc := range []struct {
		status int
		reason string
	}{
		{403, "KNOWLEDGE_FORBIDDEN"}, {404, "KNOWLEDGE_SPACE_NOT_FOUND"}, {409, "KNOWLEDGE_INDEX_CONFLICT"}, {422, "KNOWLEDGE_UNSUPPORTED_INGESTION_SETTING"},
	} {
		transport := &http.Client{Transport: knowledgeRoundTrip(func(r *http.Request) (*http.Response, error) {
			return knowledgeResponse(tc.status, `{"reason_code":"`+tc.reason+`","trace_id":"fixture-trace"}`), nil
		})}
		client, err := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "fixture", transport)
		require.NoError(t, err)
		_, err = client.UpsertKnowledgeDocument(context.Background(), fwknowledge.KnowledgeDocument{SpaceID: "11111111-1111-4111-8111-111111111111", Title: "fixture", URI: "plugin://fixture", Content: "fixture", ContentType: "text/plain", HostIngestion: &fwknowledge.DocumentIngestionSettings{Schema: fwknowledge.DocumentIngestionSchema}})
		require.Equal(t, tc.status, fwknowledge.HTTPStatus(err))
		require.Equal(t, fwknowledge.ErrorCode(tc.reason), fwknowledge.CodeOf(err))
	}
}
