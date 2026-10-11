package delegated

import (
	"context"
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testWSHandle = "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func wsWireIdentity(schema string) map[string]any {
	v := validIdentity()
	v.Schema = schema
	v.Transport = "ws_upgrade"
	if schema == contracts.WSIdentityResultSchema {
		v.ProofExpiresAt = v.UserSessionExpiresAt
	}
	raw, _ := json.Marshal(v)
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	fields["ws_revalidation_handle"] = testWSHandle
	fields["ws_revalidation_expires_at"] = v.UserSessionExpiresAt
	return fields
}
func TestWSTypedClientUsesServiceCredentialAndHidesHandle(t *testing.T) {
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		t.Run(scheme, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != scheme+" server-owned" || r.URL.Path != "/api/v1/tenant/invocations" || r.Method != "POST" || r.URL.RawQuery != "" {
					t.Error("incorrect WS identity transport")
				}
				var request struct {
					CapabilityID      string `json:"capability_id"`
					PreferredProtocol string `json:"preferred_protocol"`
					Payload           struct {
						Body struct {
							Operation string            `json:"operation"`
							Request   map[string]string `json:"request"`
						} `json:"body"`
					} `json:"payload"`
				}
				_ = json.NewDecoder(r.Body).Decode(&request)
				if request.CapabilityID != contracts.IdentityContextCapability || request.PreferredProtocol != "core_internal" || request.Payload.Body.Operation != "revalidate_ws_identity" || len(request.Payload.Body.Request) != 2 || request.Payload.Body.Request["schema"] != contracts.WSIdentityQuerySchema || request.Payload.Body.Request["ws_revalidation_handle"] != testWSHandle {
					t.Error("WS typed contract mismatch")
				}
				writeIdentity(w, wsWireIdentity(contracts.WSIdentityResultSchema))
			}))
			defer server.Close()
			cfg := CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "server-owned", nil })}
			client, _ := NewCoreClient(cfg)
			if scheme == "ApiKey" {
				client, _ = NewCoreClientWithAPIKey(cfg, "server-owned")
			}
			out, err := client.RevalidateWSIdentity(context.Background(), contracts.WSIdentityQuery{Handle: testWSHandle})
			if err != nil {
				t.Fatal(err)
			}
			if out.WSRevalidationHandle != testWSHandle || !out.ProofExpiresAt.Equal(out.UserSessionExpiresAt) {
				t.Fatal("handle deadline mapping lost")
			}
			wire, _ := json.Marshal(out)
			if strings.Contains(string(wire), testWSHandle) {
				t.Fatal("server handle leaked in serialized result")
			}
		})
	}
}
func TestWSTypedClientRejectsMissingFieldsAndWrongResponseSchema(t *testing.T) {
	for _, field := range []string{"ws_revalidation_handle", "ws_revalidation_expires_at", "schema"} {
		t.Run(field, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fields := wsWireIdentity(contracts.WSIdentityResultSchema)
				delete(fields, field)
				writeIdentity(w, fields)
			}))
			defer server.Close()
			client, _ := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin}, "server-owned")
			if _, err := client.RevalidateWSIdentity(context.Background(), contracts.WSIdentityQuery{Handle: testWSHandle}); iamerrors.CodeOf(err) != iamerrors.CodeIdentityResponseInvalid {
				t.Fatal("incomplete WS result accepted")
			}
		})
	}
}
func TestResolveWSHandleRemainsServerOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeIdentity(w, wsWireIdentity(contracts.IdentityResultSchema))
	}))
	defer server.Close()
	client, _ := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin}, "server-owned")
	out, err := client.ResolveDelegatedIdentity(context.Background(), contracts.DelegatedIdentityQuery{ContextTicket: identityTicket})
	if err != nil {
		t.Fatal(err)
	}
	if out.WSRevalidationHandle != testWSHandle || out.WSRevalidationExpiresAt == nil {
		t.Fatal("WS handle not retained")
	}
	wire, _ := json.Marshal(out)
	if strings.Contains(string(wire), testWSHandle) {
		t.Fatal("handle exposed to browser serialization")
	}
}
