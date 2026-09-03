package knowledge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/stretchr/testify/require"
)

func TestClientUsesKnowledgeQABridgeContract(t *testing.T) {
	transport := knowledgeRoundTrip(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer service-token", req.Header.Get("Authorization"))
		switch req.URL.Path {
		case "/api/v1/openapi/knowledge-spaces/qa/retrieval-plan":
			return knowledgeResponse(http.StatusOK, `{"data":{"tenant_uuid":"7c50b003-da2f-4514-aa4a-6f095fc4dc6b","intent":"answer","candidateSpaces":[{"spaceId":"4ef5fb5e-97e4-4db6-a683-50e590589716","spaceName":"FAQ","strategy":"semantic","citationCoverage":1}],"sessionId":"session-001"}}`), nil
		case "/api/v1/openapi/knowledge-spaces/qa/memory-snapshot":
			return knowledgeResponse(http.StatusOK, `{"data":{"tenant_uuid":"7c50b003-da2f-4514-aa4a-6f095fc4dc6b","sessionId":"session-001","citations":[{"chunkId":"chunk-001","spaceId":"4ef5fb5e-97e4-4db6-a683-50e590589716","status":"active"}]}}`), nil
		default:
			t.Fatalf("unexpected request path: %s", req.URL.Path)
			return nil, nil
		}
	})
	client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) {
		return "service-token", nil
	}), &http.Client{Transport: transport})
	require.NoError(t, err)

	plan, err := client.RetrievalPlan(context.Background(), RetrievalPlanInput{Intent: "answer", SessionID: "session-001"})
	require.NoError(t, err)
	require.Equal(t, "FAQ", plan.CandidateSpaces[0].SpaceName)
	snapshot, err := client.UpsertMemorySnapshot(context.Background(), MemorySnapshotInput{SessionID: "session-001"})
	require.NoError(t, err)
	require.Equal(t, "chunk-001", snapshot.Citations[0].ChunkID)
}

func TestClientReturnsTypedHTTPError(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) {
		return knowledgeResponse(http.StatusForbidden, `{"reason_code":"KNOWLEDGE_FORBIDDEN"}`), nil
	})})
	require.NoError(t, err)

	_, err = client.RetrievalPlan(context.Background(), RetrievalPlanInput{Intent: "answer"})
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, http.StatusForbidden, httpErr.StatusCode)
}

func TestClientUsesTenantKnowledgeHostContract(t *testing.T) {
	transport := knowledgeRoundTrip(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer service-token", req.Header.Get("Authorization"))
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/tenant/knowledge/spaces":
			return knowledgeResponse(http.StatusOK, `{"data":{"items":[{"space_uuid":"11111111-1111-1111-1111-111111111111","name":"FAQ","status":"active"}]}}`), nil
		case "POST /api/v1/tenant/knowledge/search":
			return knowledgeResponse(http.StatusOK, `{"data":{"items":[{"space_uuid":"11111111-1111-1111-1111-111111111111","document_uuid":"22222222-2222-2222-2222-222222222222","title":"Refund FAQ","uri":"powerx://faq/refund","excerpt":"Refund policy","tags":["refund"]}]}}`), nil
		case "POST /api/v1/tenant/knowledge/spaces/11111111-1111-1111-1111-111111111111/documents":
			return knowledgeResponse(http.StatusAccepted, `{"data":{"job_uuid":"33333333-3333-3333-3333-333333333333","status":"queued","operation":"upsert","document_uuid":"22222222-2222-2222-2222-222222222222"}}`), nil
		case "DELETE /api/v1/tenant/knowledge/spaces/11111111-1111-1111-1111-111111111111/documents/22222222-2222-2222-2222-222222222222":
			return knowledgeResponse(http.StatusAccepted, `{"data":{"job_uuid":"44444444-4444-4444-4444-444444444444","status":"queued","operation":"delete","document_uuid":"22222222-2222-2222-2222-222222222222"}}`), nil
		case "POST /api/v1/tenant/knowledge/spaces/11111111-1111-1111-1111-111111111111/indexes:rebuild":
			return knowledgeResponse(http.StatusAccepted, `{"data":{"job_uuid":"55555555-5555-5555-5555-555555555555","status":"queued","operation":"rebuild"}}`), nil
		case "GET /api/v1/tenant/knowledge/index-jobs/55555555-5555-5555-5555-555555555555":
			return knowledgeResponse(http.StatusOK, `{"data":{"job_uuid":"55555555-5555-5555-1111-555555555555","space_uuid":"11111111-1111-1111-1111-111111111111","status":"succeeded","operation":"rebuild"}}`), nil
		default:
			t.Fatalf("unexpected host request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})
	client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) {
		return "service-token", nil
	}), &http.Client{Transport: transport})
	require.NoError(t, err)

	spaces, err := client.ListKnowledgeSpaces(context.Background(), fwknowledge.ListSpacesInput{TenantUUID: "tenant-a"})
	require.NoError(t, err)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", spaces[0].SpaceID)

	result, err := client.SearchKnowledge(context.Background(), fwknowledge.KnowledgeQuery{Query: "refund", TenantUUID: "tenant-a", Visibility: fwknowledge.VisibilityTenant})
	require.NoError(t, err)
	require.Equal(t, "Refund policy", result.Chunks[0].Text)
	require.Equal(t, "22222222-2222-2222-2222-222222222222", result.Chunks[0].Citation.DocumentID)

	upsert, err := client.UpsertKnowledgeDocument(context.Background(), fwknowledge.KnowledgeDocument{SpaceID: spaces[0].SpaceID, Title: "Refund FAQ", URI: "powerx://faq/refund", Content: "Refund policy", ContentType: "text/markdown", Version: "v1"})
	require.NoError(t, err)
	require.Equal(t, "22222222-2222-2222-2222-222222222222", upsert.DocumentID)

	_, err = client.DeleteKnowledgeDocument(context.Background(), fwknowledge.DeleteDocumentInput{SpaceID: spaces[0].SpaceID, DocumentID: upsert.DocumentID})
	require.NoError(t, err)
	rebuild, err := client.ReindexKnowledgeDocument(context.Background(), fwknowledge.ReindexInput{SpaceID: spaces[0].SpaceID})
	require.NoError(t, err)
	require.Equal(t, fwknowledge.IndexOperationReindex, rebuild.Operation)
	job, err := client.GetKnowledgeIndexJob(context.Background(), fwknowledge.IndexJobQuery{JobID: rebuild.JobID, TenantUUID: "tenant-a"})
	require.NoError(t, err)
	require.Equal(t, fwknowledge.IndexStatusSucceeded, job.Status)
}

func TestClientRejectsUnsupportedHostQueryFilters(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, nil)
	require.NoError(t, err)
	_, err = client.SearchKnowledge(context.Background(), fwknowledge.KnowledgeQuery{Query: "refund", Tags: []string{"refund"}})
	require.Equal(t, fwknowledge.CodeUnsupportedCapability, fwknowledge.CodeOf(err))
	_, err = client.ReindexKnowledgeDocument(context.Background(), fwknowledge.ReindexInput{SpaceID: "space-a", DocumentID: "document-a"})
	require.Equal(t, fwknowledge.CodeUnsupportedCapability, fwknowledge.CodeOf(err))
}

func TestClientMapsKnowledgeHostErrorEnvelope(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) {
		return knowledgeResponse(http.StatusForbidden, `{"error_code":"KNOWLEDGE_FORBIDDEN","reason_code":"KNOWLEDGE_FORBIDDEN"}`), nil
	})})
	require.NoError(t, err)
	_, err = client.ListKnowledgeSpaces(context.Background(), fwknowledge.ListSpacesInput{})
	require.Equal(t, fwknowledge.CodeForbidden, fwknowledge.CodeOf(err))
}

type knowledgeRoundTrip func(*http.Request) (*http.Response, error)

func (f knowledgeRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func knowledgeResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
