package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientUsesTypedAIEndpointsAndResponseEnvelope(t *testing.T) {
	transport := &recordingTransport{}
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "test-token"}, &http.Client{Transport: transport})
	require.NoError(t, err)

	llm, err := client.LLMInvoke(context.Background(), LLMInvokeInput{
		ModelKey: "gpt-test",
		Inputs:   []ContentItem{{Role: "user", Type: "text", Content: "hello"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello back", llm.Text)
	require.Equal(t, "stop", llm.FinishReason)

	models, err := client.ListLLMModels(context.Background(), "openai")
	require.NoError(t, err)
	require.Equal(t, "tenant-prod", models.Environment)
	require.Equal(t, "gpt-test", models.Items[0].ModelKey)

	session, err := client.CreateLLMSession(context.Background(), CreateLLMSessionInput{ModelKey: "gpt-test"})
	require.NoError(t, err)
	require.Equal(t, "4e5b9a11-a9f4-4253-a0e8-f1eed119c5f4", session.SessionID)
	require.NoError(t, client.AppendLLMSessionMessage(context.Background(), session.SessionID, AppendLLMSessionMessageInput{
		Role: "user", Content: []ContentItem{{Type: "text", Content: "follow up"}},
	}))

	embedding, err := client.EmbeddingInvoke(context.Background(), EmbeddingInvokeInput{ModelKey: "embed-test", Inputs: []string{"text"}})
	require.NoError(t, err)
	require.Equal(t, float32(0.1), embedding.Vectors[0][0])

	image, err := client.ImageInvoke(context.Background(), ModalInvokeInput{ModelKey: "image-test", Inputs: []ContentItem{{Type: "text", Content: "a tree"}}})
	require.NoError(t, err)
	require.JSONEq(t, `{"url":"https://media.example/tree.png"}`, string(image.Data))

	require.Equal(t, []string{
		"POST /api/v1/ai/llm/invoke",
		"GET /api/v1/ai/llm/models?provider=openai",
		"POST /api/v1/ai/llm/sessions",
		"POST /api/v1/ai/llm/sessions/4e5b9a11-a9f4-4253-a0e8-f1eed119c5f4/messages",
		"POST /api/v1/ai/embedding/invoke",
		"POST /api/v1/ai/image/invoke",
	}, transport.calls)
	for _, authorization := range transport.authorizations {
		require.Equal(t, "Bearer test-token", authorization)
	}
}

func TestClientReturnsHTTPErrorWithoutFabricatingData(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "test-token"}, &http.Client{Transport: failingTransport{}})
	require.NoError(t, err)

	_, err = client.VLMInvoke(context.Background(), ModalInvokeInput{ModelKey: "vlm-test"})
	require.Error(t, err)
	apiErr, ok := err.(*HTTPError)
	require.True(t, ok)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
}

type recordingTransport struct {
	calls          []string
	authorizations []string
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls = append(t.calls, req.Method+" "+req.URL.RequestURI())
	t.authorizations = append(t.authorizations, req.Header.Get("Authorization"))
	data := `{"data":{}}`
	switch req.URL.Path {
	case "/api/v1/ai/llm/invoke":
		data = `{"data":{"output":{"type":"text","text":"hello back"},"meta":{"finish_reason":"stop"},"usage":{"total_tokens":2}}}`
	case "/api/v1/ai/llm/models":
		data = `{"data":{"env":"tenant-prod","items":[{"model_key":"gpt-test","provider":"openai","model":"gpt-test","label":"GPT Test","source":"profile","configured":true,"profile_configured":true}]}}`
	case "/api/v1/ai/llm/sessions":
		data = `{"data":{"session_id":"4e5b9a11-a9f4-4253-a0e8-f1eed119c5f4"}}`
	case "/api/v1/ai/embedding/invoke":
		data = `{"data":{"vectors":[[0.1,0.2]]}}`
	case "/api/v1/ai/image/invoke":
		data = `{"data":{"url":"https://media.example/tree.png"}}`
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(data)), Request: req}, nil
}

type failingTransport struct{}

func (failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":"unavailable"}`)), Request: req}, nil
}
