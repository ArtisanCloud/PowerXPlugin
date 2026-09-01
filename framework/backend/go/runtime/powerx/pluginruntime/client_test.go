package pluginruntime

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientUsesTenantRuntimeRoutesAndUUIDObjects(t *testing.T) {
	transport := &runtimeTransport{}
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: transport})
	require.NoError(t, err)

	spaces, err := client.ListKnowledgeSpaces(context.Background(), ListKnowledgeSpacesInput{Page: 2, PageSize: 10, Status: "published", Keyword: "marketing"})
	require.NoError(t, err)
	require.Equal(t, "bd29fd64-c51c-46ca-9827-a9deff6cae35", spaces.Items[0].UUID)
	require.Equal(t, int64(1), spaces.Total)

	created, err := client.InstantiateAgent(context.Background(), InstantiateAgentInput{Name: "Marketing Agent", SkillIDs: []string{"skill-a"}})
	require.NoError(t, err)
	require.Equal(t, "bc56656a-c523-4881-8e6d-385881cf94d4", created.UUID)

	agents, err := client.ListAgents(context.Background(), "prod", "published")
	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, "bc56656a-c523-4881-8e6d-385881cf94d4", agents[0].UUID)

	require.Equal(t, []string{
		"GET /api/v1/tenant/plugin-runtime/knowledge-spaces?keyword=marketing&page=2&page_size=10&status=published",
		"POST /api/v1/tenant/plugin-runtime/agents/instantiate",
		"GET /api/v1/tenant/plugin-runtime/agents?env=prod&status=published",
	}, transport.calls)
	for _, authorization := range transport.authorizations {
		require.Equal(t, "Bearer service-token", authorization)
	}
}

type runtimeTransport struct {
	calls          []string
	authorizations []string
}

func (t *runtimeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls = append(t.calls, req.Method+" "+req.URL.RequestURI())
	t.authorizations = append(t.authorizations, req.Header.Get("Authorization"))
	payload := `{"data":{}}`
	switch req.URL.Path {
	case "/api/v1/tenant/plugin-runtime/knowledge-spaces":
		payload = `{"data":{"items":[{"uuid":"bd29fd64-c51c-46ca-9827-a9deff6cae35","space_name":"Marketing","status":"published"}],"pagination":{"total":1,"page":2,"page_size":10}}}`
	case "/api/v1/tenant/plugin-runtime/agents/instantiate":
		payload = `{"data":{"uuid":"bc56656a-c523-4881-8e6d-385881cf94d4","key":"marketing-agent","name":"Marketing Agent","env":"prod","status":"draft"}}`
	case "/api/v1/tenant/plugin-runtime/agents":
		payload = `{"data":{"items":[{"uuid":"bc56656a-c523-4881-8e6d-385881cf94d4","key":"marketing-agent","name":"Marketing Agent","env":"prod","status":"published"}],"count":1}}`
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
}
