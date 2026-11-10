package hostcontract

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

//go:embed testdata/plugin-service-errors-stream.v1.json
var serviceFixture []byte

func TestInvocationFailureSafeRecoveryAndSeparateTrace(t *testing.T) {
	var f struct {
		Cases []struct {
			Name   string          `json:"name"`
			Status int             `json:"status"`
			Body   json.RawMessage `json:"body"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(serviceFixture, &f); err != nil {
		t.Fatal(err)
	}
	for _, test := range f.Cases {
		t.Run(test.Name, func(t *testing.T) {
			err := DecodeInvocationFailure(test.Status, nil, test.Body, "TEST_UPSTREAM")
			if err.ErrorCode != test.Name || err.ReasonCode != test.Name || err.StatusCode != test.Status || err.TraceID != "11111111-1111-4111-8111-111111111111" || err.RequestID != "33333333-3333-4333-8333-333333333333" {
				t.Fatal("FAILURE_METADATA_LOST", err)
			}
			if test.Name == "MESSAGING_IDEMPOTENCY_CONFLICT" {
				if err.Details.Retryable == nil || *err.Details.Retryable || err.Details.CurrentRevision == nil || *err.Details.CurrentRevision != 0 || err.Details.NextActionCode != "inspect_existing_request" || err.Details.UpstreamTraceID == err.TraceID || err.Details.UpstreamTraceID != "22222222-2222-4222-8222-222222222222" {
					t.Fatal("RECOVERY_VALUES_LOST", err.Details)
				}
			}
			if test.Name == "CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH" && (err.Details.ExpectedProviderVersion != "1.2.3" || err.Details.ActualProviderVersion != "1.2.2") {
				t.Fatal("RUNTIME_VERSIONS_LOST")
			}
		})
	}
}

func TestInvocationFailureRejectsPrivateOrInvalidRecoveryValues(t *testing.T) {
	raw := []byte(`{"error_code":"MESSAGING_TEMPORARILY_UNAVAILABLE","reason_code":"MESSAGING_TEMPORARILY_UNAVAILABLE","message":"private-body","details":{"trace_id":"core-trace","raw":"private-body","next_action_code":"bad action","retryable":"true","retry_after_ms":-1,"retry_after":86401,"current_revision":null,"expected_revision":9007199254740992,"operation_uuid":"not-a-uuid","upstream_request_id":"not-a-uuid","upstream_trace_id":"private-body","recovery":{"next_action_code":"check_delivery","retryable":false,"retry_after_ms":0,"current_revision":9007199254740991,"raw":"private-body","recovery":{"retryable":true}}}}`)
	e := DecodeInvocationFailure(503, http.Header{"X-Trace-Id": []string{"header-trace"}, "X-Request-Id": []string{"header-request"}}, raw, "TEST_UPSTREAM")
	d := e.Details
	if d.NextActionCode != "" || d.Retryable != nil || d.RetryAfterMS != nil || d.RetryAfter != nil || d.CurrentRevision != nil || d.ExpectedRevision != nil || d.OperationUUID != "" || d.UpstreamRequestID != "" || d.UpstreamTraceID != "" {
		t.Fatal("INVALID_RECOVERY_ACCEPTED", d)
	}
	if e.TraceID != "header-trace" || e.RequestID != "header-request" || d.TraceID != "core-trace" || d.Recovery == nil || d.Recovery.Retryable == nil || *d.Recovery.Retryable || *d.Recovery.RetryAfterMS != 0 || *d.Recovery.CurrentRevision != 9007199254740991 {
		t.Fatal("VALID_RECOVERY_LOST", e)
	}
	encoded, _ := json.Marshal(e)
	if strings.Contains(string(encoded), "private-body") || strings.Contains(e.Error(), "private-body") {
		t.Fatal("PRIVATE_RESPONSE_LEAKED")
	}
	for _, body := range []string{`{"error_code":"GOOD_CODE"} trailing`, `not-json`, `null`, `{"message":"PRIVATE_BODY"}`} {
		failure := DecodeInvocationFailure(502, nil, []byte(body), "TEST_UPSTREAM")
		if failure.ReasonCode != "TEST_UPSTREAM" {
			t.Fatal("FREE_TEXT_INTERPRETED_AS_CODE", failure)
		}
	}
}
