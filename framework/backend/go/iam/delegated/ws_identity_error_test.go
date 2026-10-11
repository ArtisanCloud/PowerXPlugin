package delegated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func TestWSRevalidationPreservesErrorPlaneWithoutLeakingHandle(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		for _, failure := range []struct {
			status int
			code   string
		}{
			{401, "IAM_CONTEXT_USER_SESSION_INVALID"},
			{403, "IAM_CONTEXT_USER_FORBIDDEN"},
			{403, "IAM_CONTEXT_SERVICE_FORBIDDEN"},
			{409, "IAM_CONTEXT_POLICY_CHANGED"},
			{503, "IAM_CONTEXT_UNAVAILABLE"},
		} {
			t.Run(scheme+"/"+failure.code, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Header.Get("Authorization") != scheme+" fixture-service" {
						t.Error("AUTH_SCHEME_MISMATCH")
					}
					var body struct {
						Payload struct {
							Body struct {
								Operation string `json:"operation"`
							} `json:"body"`
						} `json:"payload"`
					}
					if json.NewDecoder(r.Body).Decode(&body) != nil || body.Payload.Body.Operation != "revalidate_ws_identity" {
						t.Error("OPERATION_MISMATCH")
					}
					w.Header().Set("X-Trace-ID", "ws-trace")
					w.Header().Set("X-Request-ID", "ws-request")
					w.WriteHeader(failure.status)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": failure.status, "error_code": failure.code, "reason_code": failure.code, "message": testWSHandle, "source_trace_id": "ws-source-trace", "source_request_id": "ws-source-request"})
				}))
				defer server.Close()
				config := CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "fixture-service", nil })}
				client, err := NewCoreClient(config)
				if scheme == "ApiKey" {
					client, err = NewCoreClientWithAPIKey(config, "fixture-service")
				}
				if err != nil {
					t.Fatal(err)
				}
				result, err := client.RevalidateWSIdentity(context.Background(), contracts.WSIdentityQuery{Handle: testWSHandle})
				var detail *IdentityRequestError
				if result != nil || !errors.As(err, &detail) || detail.Status != failure.status || iamerrors.CodeOf(err) != failure.code || detail.TraceID != "ws-trace" || detail.RequestID != "ws-request" || detail.SourceTraceID != "ws-source-trace" || detail.SourceRequestID != "ws-source-request" || calls != 1 {
					t.Fatal("ERROR_METADATA_MISMATCH")
				}
				if strings.Contains(fmt.Sprintf("%+v", err), testWSHandle) {
					t.Fatal("WS_HANDLE_EXPOSED")
				}
			})
		}
	}
}
func TestWSRevalidationQueryCannotExposeHandle(t *testing.T) {
	query := contracts.WSIdentityQuery{Handle: testWSHandle}
	raw, err := json.Marshal(query)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), testWSHandle) || strings.Contains(fmt.Sprintf("%+v", query), testWSHandle) || strings.Contains(fmt.Sprintf("%#v", query), testWSHandle) {
		t.Fatal("WS_HANDLE_EXPOSED")
	}
}
