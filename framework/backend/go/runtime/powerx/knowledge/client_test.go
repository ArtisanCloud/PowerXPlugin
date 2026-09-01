package knowledge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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

type knowledgeRoundTrip func(*http.Request) (*http.Response, error)

func (f knowledgeRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func knowledgeResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
