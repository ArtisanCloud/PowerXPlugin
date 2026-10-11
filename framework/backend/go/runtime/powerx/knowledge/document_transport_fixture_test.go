package knowledge

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/url"
	"testing"
)

// Reuse business fixtures while asserting the actual delegated Registry envelope.
func documentTransportFixture(t *testing.T, scheme string, handler func(*http.Request) (*http.Response, error)) knowledgeRoundTrip {
	return func(req *http.Request) (*http.Response, error) {
		if scheme == "ApiKey" {
			return handler(req)
		}
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/tenant/invocations", req.URL.Path)
		var envelope struct {
			CapabilityID      string `json:"capability_id"`
			PreferredProtocol string `json:"preferred_protocol"`
			Payload           struct {
				Method   string          `json:"method"`
				Endpoint string          `json:"endpoint"`
				Body     json.RawMessage `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&envelope))
		require.Equal(t, documentManageCapability, envelope.CapabilityID)
		require.Equal(t, "core_internal", envelope.PreferredProtocol)
		require.Equal(t, "INVOKE", envelope.Payload.Method)
		require.Equal(t, "core://knowledge/documents", envelope.Payload.Endpoint)
		var op struct {
			Operation    string          `json:"operation"`
			SpaceUUID    string          `json:"space_uuid"`
			DocumentUUID string          `json:"document_uuid"`
			JobUUID      string          `json:"job_uuid"`
			Document     json.RawMessage `json:"document"`
			Rebuild      json.RawMessage `json:"rebuild"`
		}
		require.NoError(t, json.Unmarshal(envelope.Payload.Body, &op))
		fake := req.Clone(req.Context())
		fake.URL = &url.URL{Path: "/api/v1/tenant/knowledge"}
		field := "job"
		switch op.Operation {
		case "submit_document":
			fake.URL.Path += "/spaces/" + op.SpaceUUID + "/documents"
			fake.Body = io.NopCloser(bytes.NewReader(op.Document))
		case "rebuild_space", "rebuild_document":
			fake.URL.Path += "/spaces/" + op.SpaceUUID
			if op.Operation == "rebuild_document" {
				require.NotEmpty(t, op.DocumentUUID)
				fake.URL.Path += "/documents/" + op.DocumentUUID
			} else {
				require.Empty(t, op.DocumentUUID)
			}
			fake.URL.Path += "/indexes:rebuild"
			fake.Body = io.NopCloser(bytes.NewReader(op.Rebuild))
		case "get_job":
			fake.Method = http.MethodGet
			fake.URL.Path += "/index-jobs/" + op.JobUUID
		case "get_job_chunks":
			fake.Method = http.MethodGet
			fake.URL.Path += "/index-jobs/" + op.JobUUID + "/chunks"
			field = ""
		default:
			t.Fatalf("unexpected operation %s", op.Operation)
		}
		resp, err := handler(fake)
		if err != nil || resp.StatusCode >= 400 {
			return resp, err
		}
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		var original struct {
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &original))
		business := original.Data
		if field != "" {
			encoded, err := json.Marshal(map[string]json.RawMessage{field: business})
			require.NoError(t, err)
			business = encoded
		}
		encoded, err := json.Marshal(map[string]any{"data": map[string]any{"payload": json.RawMessage(business), "trace_id": "invocation-fixture"}, "request_id": "request-fixture"})
		require.NoError(t, err)
		resp.Body = io.NopCloser(bytes.NewReader(encoded))
		return resp, nil
	}
}
