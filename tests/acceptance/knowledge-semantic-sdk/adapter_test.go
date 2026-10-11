package hostsnapshot_test

import (
	"bytes"
	"context"
	"encoding/json"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	sdk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/knowledge"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type sdkTap struct {
	name    string
	records *[]evidenceRecord
	called  *bool
}

func (tap sdkTap) RoundTrip(req *http.Request) (*http.Response, error) {
	*tap.called = true
	response, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	trace := response.Header.Get("X-Trace-ID")
	if trace == "" {
		trace = req.Header.Get("X-Trace-ID")
	}
	var decoded map[string]json.RawMessage
	_ = json.Unmarshal(raw, &decoded)
	requestID := decoded["request_id"]
	// Body is actual controlled Core output; credentials are never recorded.
	*tap.records = append(*tap.records, evidenceRecord{Name: "framework_sdk:" + tap.name, HTTP: response.StatusCode, Trace: trace, Response: raw})
	_ = requestID
	return response, nil
}
func convert(body, out any) bool {
	raw, err := json.Marshal(body)
	if err != nil {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out) == nil
}
func frameworkCall(t *testing.T, name, base, method, path, auth string, body any, unknown bool, httpClient *http.Client, records *[]evidenceRecord) (bool, int, map[string]json.RawMessage) {
	if unknown || !strings.HasPrefix(path, "/api/v1/tenant/knowledge/") {
		return false, 0, nil
	}
	if os.Getenv("POWERX_SDK_APIKEY_DOCUMENT_PASS") == "1" && strings.HasPrefix(auth, "Bearer ") && (strings.Contains(path, "/documents") || strings.Contains(path, "/index-jobs/") || strings.HasSuffix(path, "/indexes:rebuild")) {
		auth = "ApiKey " + fixtureDocumentAPIKey
		name = "explicit_api_key_document_pass:" + name
	}
	called := false
	transport := &http.Client{Timeout: httpClient.Timeout, Transport: sdkTap{name: name, records: records, called: &called}}
	var client *sdk.Client
	var err error
	if strings.HasPrefix(auth, "ApiKey ") {
		client, err = sdk.NewClientWithAPIKey(sdk.Config{BaseURL: base}, strings.TrimPrefix(auth, "ApiKey "), transport)
	} else if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		client, err = sdk.NewClientWithTokenProvider(sdk.Config{BaseURL: base}, sdk.TokenProviderFunc(func(context.Context) (string, error) { return token, nil }), transport)
	} else {
		return false, 0, nil
	}
	require.NoError(t, err)
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/tenant/knowledge/"), "/")
	var result any
	status := 200
	switch {
	case path == "/api/v1/tenant/knowledge/retrieval/query" && method == "POST":
		var in fw.SemanticQuery
		if !convert(body, &in) {
			return false, 0, nil
		}
		result, err = client.QuerySemanticKnowledge(context.Background(), in)
	case len(parts) == 3 && parts[0] == "spaces" && parts[2] == "semantic-index":
		if method == "GET" {
			result, err = client.GetSemanticCapabilities(context.Background(), parts[1])
		} else {
			var in fw.SemanticConfigureInput
			if !convert(body, &in) {
				return false, 0, nil
			}
			result, err = client.ConfigureSemanticIndex(context.Background(), parts[1], in)
		}
	case len(parts) == 3 && parts[0] == "spaces" && parts[2] == "documents" && method == "POST":
		var in struct {
			Title          string                        `json:"title"`
			URI            string                        `json:"uri"`
			Content        string                        `json:"content"`
			ContentType    string                        `json:"content_type"`
			Checksum       string                        `json:"checksum"`
			Version        string                        `json:"version"`
			Tags           []string                      `json:"tags"`
			Indexing       *fw.SemanticIndexingSettings  `json:"indexing"`
			Artifacts      []fw.SemanticArtifact         `json:"artifacts"`
			ExternalRef    *fw.SemanticExternalRef       `json:"external_ref"`
			IdempotencyKey string                        `json:"idempotency_key"`
			Ingestion      *fw.DocumentIngestionSettings `json:"ingestion"`
		}
		if !convert(body, &in) {
			return false, 0, nil
		}
		result, err = client.UpsertKnowledgeDocument(context.Background(), fw.KnowledgeDocument{SpaceID: parts[1], Title: in.Title, URI: in.URI, Content: in.Content, ContentType: in.ContentType, Checksum: in.Checksum, Version: in.Version, Tags: in.Tags, Indexing: in.Indexing, Artifacts: in.Artifacts, ExternalRef: in.ExternalRef, IdempotencyKey: in.IdempotencyKey, HostIngestion: in.Ingestion})
		status = 202
	case len(parts) >= 3 && parts[0] == "spaces" && strings.HasSuffix(path, "/indexes:rebuild") && method == "POST":
		var in fw.DocumentRebuildSettings
		if !convert(body, &in) {
			return false, 0, nil
		}
		if in.IdempotencyKey == "" {
			in.IdempotencyKey = "framework-fixture-" + name
		}
		document := ""
		if len(parts) == 5 && parts[2] == "documents" {
			document = parts[3]
		}
		result, err = client.ReindexKnowledgeDocument(context.Background(), fw.ReindexInput{SpaceID: parts[1], DocumentID: document, HostRebuild: &in})
		status = 202
	case len(parts) == 5 && parts[0] == "spaces" && parts[2] == "documents" && parts[4] == "visibility":
		var in fw.SemanticVisibilityInput
		if !convert(body, &in) {
			return false, 0, nil
		}
		result, err = client.SetDocumentVisibility(context.Background(), parts[1], parts[3], in)
	case len(parts) == 2 && parts[0] == "index-jobs":
		result, err = client.GetKnowledgeIndexJob(context.Background(), fw.IndexJobQuery{JobID: parts[1]})
	case len(parts) == 3 && parts[0] == "index-jobs" && parts[2] == "chunks":
		var chunks []fw.IndexJobChunk
		chunks, err = client.GetKnowledgeIndexJobChunks(context.Background(), fw.IndexJobQuery{JobID: parts[1]})
		result = map[string]any{"items": chunks}
	default:
		return false, 0, nil
	}
	if err != nil {
		status = fw.HTTPStatus(err)
		response := map[string]any{"reason_code": fw.CodeOf(err)}
		raw, _ := json.Marshal(response)
		if !called {
			*records = append(*records, evidenceRecord{Name: "framework_local_validation:" + name, HTTP: 0, Response: json.RawMessage(`{"client_status":` + stringMustJSON(t, status) + `,"reason_code":"` + string(fw.CodeOf(err)) + `"}`)})
		}
		return true, status, map[string]json.RawMessage{"reason_code": json.RawMessage(`"` + string(fw.CodeOf(err)) + `"`), "error": raw}
	}
	if job, ok := result.(*fw.KnowledgeIndexJob); ok {
		raw, _ := json.Marshal(job)
		var legacy map[string]any
		_ = json.Unmarshal(raw, &legacy)
		legacy["job_uuid"] = job.JobID
		legacy["space_uuid"] = job.SpaceID
		legacy["document_uuid"] = job.DocumentID
		legacy["completed_at"] = job.FinishedAt
		if job.Operation == fw.IndexOperationReindex {
			legacy["operation"] = "rebuild"
		}
		result = legacy
	}
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	return true, status, map[string]json.RawMessage{"data": raw}
}
