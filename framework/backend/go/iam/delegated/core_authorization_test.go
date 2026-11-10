package delegated

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

//go:embed testdata/plugin-authorization.v1.json
var authorizationFixture []byte

type authorizationTestFixture struct {
	Request     contracts.AuthorizationRequest `json:"request"`
	WireRequest map[string]any                 `json:"wire_request"`
	Response    json.RawMessage                `json:"response"`
	Denied      json.RawMessage                `json:"denied_response"`
	Errors      []struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"errors"`
}

func authorizationFixtureForTest(t *testing.T) authorizationTestFixture {
	t.Helper()
	var f authorizationTestFixture
	if err := json.Unmarshal(authorizationFixture, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func authorizationClient(t *testing.T, server *httptest.Server, scheme string) *CoreClient {
	t.Helper()
	cfg := CoreClientConfig{BaseURL: server.URL + "/api/v1", PluginID: "com.example.caller", Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "fixture-service-credential", nil })}
	var c *CoreClient
	var err error
	if scheme == "ApiKey" {
		c, err = NewCoreClientWithAPIKey(cfg, "fixture-service-credential")
	} else {
		c, err = NewCoreClient(cfg)
	}
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAuthorizationServiceCredentialsAndExplicitPermission(t *testing.T) {
	f := authorizationFixtureForTest(t)
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		for _, allowed := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/%t", scheme, allowed), func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != "POST" || r.URL.Path != "/api/v1/tenant/iam/authorization:check" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != scheme+" fixture-service-credential" || r.Header.Get("tenant_uuid") != "" {
						t.Error("AUTHORIZATION_TRANSPORT_INVALID")
					}
					if r.Header.Get("X-Trace-ID") != "caller-trace" || r.Header.Get("X-Request-ID") != "caller-request" {
						t.Error("AUTHORIZATION_METADATA_LOST")
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(body, f.WireRequest) {
						t.Error("AUTHORIZATION_REQUEST_MISMATCH", body)
					}
					if allowed {
						_, _ = w.Write(f.Response)
					} else {
						_, _ = w.Write(f.Denied)
					}
				}))
				defer server.Close()
				client := authorizationClient(t, server, scheme)
				ctx := WithRequestMetadata(context.Background(), RequestMetadata{TraceID: "caller-trace", RequestID: "caller-request"})
				out, err := client.Authorize(ctx, f.Request)
				if err != nil {
					t.Fatal(err)
				}
				if out.Allowed != allowed || out.PluginID != f.Request.PluginID || out.PermissionCode != f.Request.PermissionCode || out.Resource != "conversation" || out.TenantUUID != f.Request.TenantUUID || out.MemberUUID != f.Request.MemberUUID || out.UserUUID != f.Request.UserUUID || out.TraceID != "core-authorization-trace" || out.Mode != "delegated" || calls != 1 {
					t.Fatal("AUTHORIZATION_DECISION_INVALID", out)
				}
				if !allowed && out.ReasonCode != iamerrors.CodePermissionDenied {
					t.Fatal("DENIED_REASON_LOST")
				}
			})
		}
	}
}

func TestAuthorizationRejectsIncompleteAndCrossScopeDecisions(t *testing.T) {
	f := authorizationFixtureForTest(t)
	mutations := map[string]func(map[string]any){
		"allowed-missing":          func(v map[string]any) { delete(v, "allowed") },
		"allowed-null":             func(v map[string]any) { v["allowed"] = nil },
		"reason-missing":           func(v map[string]any) { delete(v, "reason_code") },
		"denied-without-reason":    func(v map[string]any) { v["allowed"] = false },
		"allow-with-denied-reason": func(v map[string]any) { v["reason_code"] = iamerrors.CodePermissionDenied },
		"unknown-denied":           func(v map[string]any) { v["allowed"] = false; v["reason_code"] = iamerrors.CodeUpstreamDependency },
		"plugin-missing":           func(v map[string]any) { delete(v, "plugin_id") },
		"cross-plugin":             func(v map[string]any) { v["plugin_id"] = "com.example.other" },
		"cross-permission":         func(v map[string]any) { v["permission_code"] = "messages.conversation:read" },
		"cross-tenant":             func(v map[string]any) { v["tenant_uuid"] = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" },
		"cross-member":             func(v map[string]any) { v["member_uuid"] = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" },
		"cross-user":               func(v map[string]any) { v["user_uuid"] = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" },
		"nil-user":                 func(v map[string]any) { v["user_uuid"] = "00000000-0000-0000-0000-000000000000" },
		"missing-resource":         func(v map[string]any) { delete(v, "resource") },
		"missing-action":           func(v map[string]any) { delete(v, "action") },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var envelope struct {
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(f.Response, &envelope); err != nil {
				t.Fatal(err)
			}
			mutate(envelope.Data)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Trace-ID", "contract-trace")
				w.Header().Set("X-Request-ID", "contract-request")
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": envelope.Data})
			}))
			defer server.Close()
			out, err := authorizationClient(t, server, "Bearer").Authorize(context.Background(), f.Request)
			var typed *AuthorizationRequestError
			if out != nil || !errors.As(err, &typed) || iamerrors.CodeOf(err) != iamerrors.CodeAuthorizationResponseInvalid || iamerrors.StatusCode(err) != 502 || typed.Status != 200 || typed.TraceID != "contract-trace" || typed.RequestID != "contract-request" {
				t.Fatal("INVALID_DECISION_ACCEPTED", out, err)
			}
		})
	}
	for _, body := range []string{`{}`, `null`, `{"code":200,"data":null}`, `{"code":200,"data":{}}`, `{"data":{"allowed":false}}`, `{"code":0,"data":{}}`, `not-json`, strings.Repeat("x", (1<<20)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) }))
		out, err := authorizationClient(t, server, "ApiKey").Authorize(context.Background(), f.Request)
		server.Close()
		if out != nil || iamerrors.CodeOf(err) != iamerrors.CodeAuthorizationResponseInvalid {
			t.Fatal("INVALID_ENVELOPE_ACCEPTED", err)
		}
	}
}

func TestAuthorizationErrorsKeepCodeStatusAndCorrelationWithoutRetry(t *testing.T) {
	f := authorizationFixtureForTest(t)
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		for _, fixture := range f.Errors {
			t.Run(fmt.Sprintf("%s/%d/%s", scheme, fixture.Status, fixture.Body), func(t *testing.T) {
				var body map[string]any
				_ = json.Unmarshal(fixture.Body, &body)
				body["message"] = "fixture-service-credential"
				body["details"] = map[string]any{"secret": "fixture-service-credential"}
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("X-Trace-ID", "header-trace")
					w.Header().Set("X-Request-ID", "header-request")
					w.WriteHeader(fixture.Status)
					_ = json.NewEncoder(w).Encode(body)
				}))
				defer server.Close()
				out, err := authorizationClient(t, server, scheme).Authorize(context.Background(), f.Request)
				var typed *AuthorizationRequestError
				if out != nil || !errors.As(err, &typed) || typed.Status != fixture.Status || typed.Code != body["error_code"] || iamerrors.CodeOf(err) != body["reason_code"] || iamerrors.StatusCode(err) != fixture.Status || typed.TraceID != "header-trace" || typed.RequestID != "header-request" || calls != 1 {
					t.Fatal("AUTHORIZATION_ERROR_LOST", out, err)
				}
				raw, _ := json.Marshal(typed)
				if strings.Contains(string(raw), "fixture-service-credential") || strings.Contains(err.Error(), "fixture-service-credential") {
					t.Fatal("ERROR_LEAKED_FREE_TEXT")
				}
			})
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"error_code":"IAM_PLUGIN_PERMISSION_NOT_REGISTERED","trace_id":"body-trace","request_id":"body-request"}`)
	}))
	defer server.Close()
	_, err := authorizationClient(t, server, "Bearer").Authorize(context.Background(), f.Request)
	var typed *AuthorizationRequestError
	if !errors.As(err, &typed) || iamerrors.CodeOf(err) != iamerrors.CodePluginPermissionNotRegistered || typed.TraceID != "body-trace" || typed.RequestID != "body-request" {
		t.Fatal("REGISTERED_PERMISSION_ERROR_MISCLASSIFIED", err)
	}
}

func TestAuthorizationValidatesBeforeHTTPAndPreservesOnlyOfficialIAMRead(t *testing.T) {
	f := authorizationFixtureForTest(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in["resource"] != "iam.member" || in["action"] != "read" || in["plugin_id"] != nil || in["permission_code"] != nil {
			t.Error("IAM_READ_WIRE_INVALID")
		}
		var envelope map[string]any
		_ = json.Unmarshal(f.Response, &envelope)
		data := envelope["data"].(map[string]any)
		delete(data, "plugin_id")
		delete(data, "permission_code")
		data["resource"] = "iam.member"
		data["action"] = "read"
		_ = json.NewEncoder(w).Encode(envelope)
	}))
	defer server.Close()
	client := authorizationClient(t, server, "Bearer")
	mutations := []func(*contracts.AuthorizationRequest){func(q *contracts.AuthorizationRequest) { q.PluginID = "" }, func(q *contracts.AuthorizationRequest) { q.PermissionCode = "" }, func(q *contracts.AuthorizationRequest) { q.UserUUID = "" }, func(q *contracts.AuthorizationRequest) { q.MemberUUID = "" }, func(q *contracts.AuthorizationRequest) { q.TenantUUID = "not-a-uuid" }, func(q *contracts.AuthorizationRequest) { q.PolicyToken = "unsupported" }, func(q *contracts.AuthorizationRequest) { q.PluginID = "*" }, func(q *contracts.AuthorizationRequest) {
		q.PluginID = ""
		q.PermissionCode = ""
		q.Resource = "messages.conversation"
		q.Action = "reply"
	}}
	for _, mutate := range mutations {
		query := f.Request
		mutate(&query)
		if _, err := client.Authorize(context.Background(), query); iamerrors.CodeOf(err) != iamerrors.CodeInvalidArgument {
			t.Fatal("INVALID_QUERY_ACCEPTED", err)
		}
	}
	if calls != 0 {
		t.Fatal("INVALID_QUERY_MADE_REQUEST")
	}
	query := f.Request
	query.PluginID = ""
	query.PermissionCode = ""
	query.Resource = "iam.member"
	query.Action = "read"
	if out, err := client.Authorize(context.Background(), query); err != nil || out.Resource != "iam.member" || calls != 1 {
		t.Fatal("OFFICIAL_IAM_READ_REJECTED", err)
	}
}

func TestAuthorizationDoesNotRedirectRetryOrSwitchCredentials(t *testing.T) {
	f := authorizationFixtureForTest(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Redirect(w, r, "/other", 307) }))
	defer server.Close()
	client := authorizationClient(t, server, "Bearer")
	tokenCalls := 0
	client.tokens = TokenProviderFunc(func(context.Context) (string, error) { tokenCalls++; return "fixture-service-credential", nil })
	if _, err := client.Authorize(context.Background(), f.Request); err == nil || calls.Load() != 1 || tokenCalls != 1 {
		t.Fatal("REDIRECT_OR_CREDENTIAL_SWITCH", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Authorize(ctx, f.Request); !errors.Is(err, context.Canceled) {
		t.Fatal("CANCELLATION_LOST", err)
	}
	if calls.Load() != 1 {
		t.Fatal("CANCELED_QUERY_MADE_REQUEST")
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer slow.Close()
	client = authorizationClient(t, slow, "ApiKey")
	client.timeout = 30 * time.Millisecond
	if _, err := client.Authorize(context.Background(), f.Request); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("TIMEOUT_LOST", err)
	}
}

func TestAuthorizationOptionalResourceActionAreComparedToCoreResponse(t *testing.T) {
	f := authorizationFixtureForTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(f.Response) }))
	defer server.Close()
	client := authorizationClient(t, server, "Bearer")
	query := f.Request
	query.Resource = "conversation"
	query.Action = "reply"
	if _, err := client.Authorize(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"resource", "action"} {
		bad := query
		if field == "resource" {
			bad.Resource = "other"
		} else {
			bad.Action = "read"
		}
		if _, err := client.Authorize(context.Background(), bad); iamerrors.CodeOf(err) != iamerrors.CodeAuthorizationResponseInvalid {
			t.Fatal("RESOURCE_ACTION_MISMATCH_ACCEPTED", err)
		}
	}
}
