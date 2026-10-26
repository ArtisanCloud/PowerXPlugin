package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// This models responses around a Core-owned registration upgrade. The SDK
// never migrates tenants, confirms upgrades or repairs bindings itself.
func TestMediaCatalogAroundLegacyTenantRegistrationUpgrade(t *testing.T) {
	var fixture struct {
		Errors []struct {
			Name   string          `json:"name"`
			Status int             `json:"status"`
			Code   string          `json:"error_code"`
			Body   json.RawMessage `json:"body"`
		} `json:"catalog_errors"`
		Upgrade struct {
			Status    int             `json:"status"`
			Body      json.RawMessage `json:"body"`
			Reason    string          `json:"expected_reason_code"`
			Published bool            `json:"machine_reason_published"`
		} `json:"upgrade_confirmation_error"`
	}
	raw, err := os.ReadFile("testdata/ai-media-v2-registration.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &fixture))
	for _, auth := range []string{"ApiKey fixture.key", "Bearer fixture.sts"} {
		t.Run(auth[:6], func(t *testing.T) {
			var stage atomic.Int32
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, auth, r.Header.Get("Authorization"))
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "core_internal", request["preferred_protocol"])
				payload := request["payload"].(map[string]any)
				require.Len(t, payload, 1)
				require.Contains(t, payload, "body")
				w.Header().Set("X-Trace-ID", "fixture.trace")
				w.Header().Set("Content-Type", "application/json")
				if int(stage.Load()) < len(fixture.Errors) {
					item := fixture.Errors[stage.Load()]
					w.WriteHeader(item.Status)
					_, _ = w.Write(item.Body)
					return
				}
				_, _ = fmt.Fprint(w, `{"data":{"protocol_used":"core_internal","fallback_used":false,"payload":{"result":{"contract_version":2,"providers":[{"provider":"fixture.provider","label":"fixture.label"}]}}}}`)
			}))
			defer server.Close()
			var client *Client
			if auth[0] == 'A' {
				client, err = NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture.key"}, server.Client())
			} else {
				client, err = NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "fixture.sts", nil }), server.Client())
			}
			require.NoError(t, err)
			for i, item := range fixture.Errors {
				if item.Name == "revoked_after_upgrade" {
					stage.Store(int32(len(fixture.Errors)))
					out, err := client.ListMediaProviders(context.Background(), ImageGeneration)
					require.NoError(t, err)
					require.Len(t, out.Providers, 1)
				}
				before := calls.Load()
				stage.Store(int32(i))
				_, err = client.ListMediaProviders(context.Background(), ImageGeneration)
				var typed *HTTPError
				require.True(t, errors.As(err, &typed))
				require.Equal(t, item.Status, typed.StatusCode)
				require.Equal(t, item.Code, typed.ErrorCode)
				require.Equal(t, "fixture.trace", typed.TraceID)
				require.Equal(t, "fixture.request", typed.RequestID)
				require.EqualValues(t, before+1, calls.Load(), "no automatic retry or fallback")
			}
			// The published upgrade reason is decoded without interpreting text.
			typed := DecodeHTTPError(fixture.Upgrade.Status, http.Header{}, fixture.Upgrade.Body)
			require.True(t, fixture.Upgrade.Published)
			require.Equal(t, "capability.upgrade_binding_mismatch", fixture.Upgrade.Reason)
			require.Equal(t, fixture.Upgrade.Reason, typed.ErrorCode)
			require.Equal(t, fixture.Upgrade.Reason, typed.ReasonCode)
			require.Equal(t, string(fixture.Upgrade.Body), typed.Body)
		})
	}
}

type upgradeErrorCase struct {
	Name   string          `json:"name"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func upgradeErrorCases(t *testing.T) []upgradeErrorCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/capability-version-upgrade-errors.v1.json")
	require.NoError(t, err)
	var fixture struct {
		Owner     string             `json:"fixture_owner"`
		Published bool               `json:"machine_reason_published"`
		Cases     []upgradeErrorCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Equal(t, "Core", fixture.Owner)
	require.True(t, fixture.Published)
	require.Len(t, fixture.Cases, 10)
	return fixture.Cases
}

func TestUpgradeErrorDecoderPreservesPublishedContract(t *testing.T) {
	for _, item := range upgradeErrorCases(t) {
		t.Run(item.Name, func(t *testing.T) {
			var body struct {
				ErrorCode  string         `json:"error_code"`
				ReasonCode string         `json:"reason_code"`
				RequestID  string         `json:"request_id"`
				Details    map[string]any `json:"details"`
			}
			require.NoError(t, json.Unmarshal(item.Body, &body))
			want := "capability.upgrade_" + item.Name
			if item.Name == "unknown" {
				want = "capability.upgrade_failed"
			}
			require.Equal(t, want, body.ErrorCode)
			require.Equal(t, want, body.ReasonCode)
			decoded := DecodeHTTPError(item.Status, http.Header{}, item.Body)
			require.Equal(t, item.Status, decoded.StatusCode)
			require.Equal(t, body.ErrorCode, decoded.ErrorCode)
			require.Equal(t, body.ReasonCode, decoded.ReasonCode)
			require.Equal(t, body.Details["trace_id"], decoded.TraceID)
			require.Equal(t, body.RequestID, decoded.RequestID)
			require.Equal(t, body.Details, decoded.Details)
			require.Equal(t, string(item.Body), decoded.Body)
			switch item.Name {
			case "runtime_apply_failed":
				require.Equal(t, true, decoded.Details["approval_persisted"])
				require.Equal(t, float64(42), decoded.Details["audit_id"])
				// Approval exists; the runtime lock outcome is still unknown.
				require.NotContains(t, decoded.Details, "runtime_confirmed")
			case "audit_finalize_failed":
				require.Equal(t, true, decoded.Details["approval_persisted"])
				require.Equal(t, true, decoded.Details["runtime_confirmed"])
				require.Equal(t, float64(42), decoded.Details["audit_id"])
			}
			withHeaders := DecodeHTTPError(item.Status, http.Header{"X-Trace-Id": {"header.trace"}, "X-Request-Id": {"header.request"}}, item.Body)
			require.Equal(t, "header.trace", withHeaders.TraceID)
			require.Equal(t, "header.request", withHeaders.RequestID)
			require.Equal(t, body.Details, withHeaders.Details)
		})
	}
}

func TestUpgradeErrorsNeverTriggerAutomaticApprovalOrRetry(t *testing.T) {
	for _, auth := range []string{"ApiKey fixture.key", "Bearer fixture.sts"} {
		t.Run(auth[:6], func(t *testing.T) {
			for _, item := range upgradeErrorCases(t) {
				t.Run(item.Name, func(t *testing.T) {
					var calls atomic.Int32
					var unexpected atomic.Int32
					// Controlled error propagation: never call the real Admin route.
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/invocations" {
							unexpected.Add(1)
						}
						require.Equal(t, auth, r.Header.Get("Authorization"))
						var request map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
						require.Equal(t, CapabilityMediaProviders, request["capability_id"])
						require.Equal(t, "core_internal", request["preferred_protocol"])
						payload := request["payload"].(map[string]any)
						require.Len(t, payload, 1)
						require.Contains(t, payload, "body")
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(item.Status)
						_, _ = w.Write(item.Body)
					}))
					defer server.Close()
					var client *Client
					var err error
					if auth[0] == 'A' {
						client, err = NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture.key"}, server.Client())
					} else {
						client, err = NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "fixture.sts", nil }), server.Client())
					}
					require.NoError(t, err)
					_, err = client.ListMediaProviders(context.Background(), ImageGeneration)
					var decoded *HTTPError
					require.ErrorAs(t, err, &decoded)
					want := DecodeHTTPError(item.Status, http.Header{}, item.Body)
					require.Equal(t, want, decoded)
					require.EqualValues(t, 1, calls.Load(), "retryable metadata cannot authorize a second request")
					require.Zero(t, unexpected.Load(), "no approval endpoint or alternate channel")
				})
			}
		})
	}
}

func TestUpgradeDiagnosticsCannotImpersonateAMachineReason(t *testing.T) {
	for _, raw := range []string{
		`{"code":500,"message":"capability.upgrade_failed","error":"capability.upgrade_binding_mismatch"}`,
		`{"code":500,"message":"capability.upgrade_binding_mismatch","error":"capability.upgrade_hash_mismatch","error_code":"capability.upgrade_failed","reason_code":"capability.upgrade_failed"}`,
	} {
		decoded := DecodeHTTPError(500, http.Header{}, []byte(raw))
		require.Equal(t, raw, decoded.Body)
		require.NotEqual(t, "capability.upgrade_binding_mismatch", decoded.ReasonCode)
		require.NotEqual(t, "capability.upgrade_hash_mismatch", decoded.ReasonCode)
	}
}
