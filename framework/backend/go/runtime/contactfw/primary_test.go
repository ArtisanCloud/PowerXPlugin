package contactfw

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

func TestPrimaryWireRequiresIndependentGrantAndExplicitExpected(t *testing.T) {
	for _, granted := range []bool{false, true} {
		t.Run(strconv.FormatBool(granted), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/v1/tenant/invocations" || r.Header.Get("Authorization") != "ApiKey fixture-key" {
					t.Error("CONTACT_PRIMARY_TRANSPORT_INVALID")
				}
				var req struct {
					CapabilityID      string `json:"capability_id"`
					PreferredProtocol string `json:"preferred_protocol"`
					Payload           struct {
						Method, Endpoint string
						Body             map[string]any
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if req.CapabilityID != CapabilityContactsPrimaryManage || req.PreferredProtocol != "core_internal" || req.Payload.Method != "INVOKE" || req.Payload.Endpoint != contactCoreEndpoint || req.Payload.Body["expected_primary_contact_uuid"] != "" || len(req.Payload.Body) != 4 {
					t.Error("CONTACT_PRIMARY_WIRE_INVALID")
				}
				w.Header().Set("Content-Type", "application/json")
				if !granted {
					w.WriteHeader(403)
					_, _ = w.Write([]byte(`{"code":403,"details":{"reason_code":"CAPABILITY_FORBIDDEN"}}`))
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"protocol_used": "core_internal", "fallback_used": false, "status": "completed", "payload": map[string]any{"payload": PrimaryContactResult{CustomerUUID: testCustomerUUID, PrimaryContactUUID: testContactUUID}}}})
			}))
			defer server.Close()
			g, err := gateway.NewClient(gateway.Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture-key"})
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: g})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = client.SetPrimary(context.Background(), SetPrimaryContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID}); CodeOf(err) != CodePrimaryExpectedRequired || calls != 0 {
				t.Fatal("CONTACT_PRIMARY_PRESENCE_INVALID", err)
			}
			expected := ""
			out, err := client.SetPrimary(context.Background(), SetPrimaryContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID, ExpectedPrimaryContactUUID: &expected})
			if granted {
				if err != nil || out == nil || out.PrimaryContactUUID != testContactUUID {
					t.Fatal("CONTACT_PRIMARY_RESULT_INVALID", err)
				}
			} else if CodeOf(err) != CodeCapabilityForbidden || HTTPStatus(err) != 403 {
				t.Fatal("CONTACT_PRIMARY_GRANT_INVALID", err)
			}
			if calls != 1 {
				t.Fatal("CONTACT_PRIMARY_CALL_COUNT_INVALID")
			}
		})
	}
}

func TestPrimaryErrorsPreserveCoreEnvelopeStatusAndTrace(t *testing.T) {
	for _, code := range []ErrorCode{CodePrimaryChanged, CodePrimaryNotActive, CodePrimaryOperationRequired, CodePrimaryChangeRequired, CodePrimaryOperationFailed, CodePrimaryExpectedRequired, CodeRequestInvalid} {
		for _, status := range []int{409, 502} {
			body, _ := json.Marshal(map[string]any{"code": status, "error_code": code, "request_id": "fixture-request", "details": map[string]any{"trace_id": "fixture-trace"}})
			upstream := &gateway.InvocationError{StatusCode: status, Body: body}
			err := mapCapabilityError(upstream)
			var typed *Error
			if !errors.As(err, &typed) || typed.Code != code || HTTPStatus(err) != status || typed.TraceID != "fixture-trace" || typed.RequestID != "fixture-request" || !errors.Is(err, upstream) {
				t.Fatal("CONTACT_PRIMARY_ERROR_METADATA_INVALID", err)
			}
		}
	}
}

func TestPrimaryRejectsIncorrectProtocolFallbackAndOwnership(t *testing.T) {
	expected := testContactUUID
	in := SetPrimaryContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID, ExpectedPrimaryContactUUID: &expected}
	for _, field := range []string{"protocol_used", "fallback_used", "status", "customer_uuid", "primary_contact_uuid", "previous_primary_contact_uuid", "payload"} {
		t.Run(field, func(t *testing.T) {
			result := map[string]any{"customer_uuid": testCustomerUUID, "primary_contact_uuid": testContactUUID, "previous_primary_contact_uuid": expected}
			data := map[string]any{"protocol_used": "core_internal", "fallback_used": false, "status": "completed", "payload": map[string]any{"payload": result}}
			switch field {
			case "protocol_used":
				data[field] = "http"
			case "fallback_used":
				data[field] = true
			case "status":
				data[field] = "running"
			case "payload":
				data[field] = result
			default:
				result[field] = "33333333-3333-4333-8333-333333333333"
			}
			client, err := NewCapabilityClient(CapabilityClientConfig{Invoker: &contactGatewayStub{response: &gateway.Response{Data: data}}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.SetPrimary(context.Background(), in); CodeOf(err) != CodeDelegateUnavailable {
				t.Fatal("CONTACT_PRIMARY_INVALID_RESPONSE_ACCEPTED", err)
			}
		})
	}
}

func TestPrimaryInputRejectsOverridesAndAbsentExpected(t *testing.T) {
	base := `{"customer_uuid":"` + testCustomerUUID + `","contact_uuid":"` + testContactUUID + `"`
	for _, suffix := range []string{`}`, `,"expected_primary_contact_uuid":null}`, `,"expected_primary_contact_uuid":"","tenant_uuid":"override"}`, `,"expected_primary_contact_uuid":"","actor":"override"}`, `,"expected_primary_contact_uuid":"","endpoint":"override"}`} {
		var input SetPrimaryContactInput
		if json.Unmarshal([]byte(base+suffix), &input) == nil {
			t.Fatal("CONTACT_PRIMARY_OVERRIDE_ACCEPTED")
		}
	}
	var input SetPrimaryContactInput
	if err := json.Unmarshal([]byte(base+`,"expected_primary_contact_uuid":""}`), &input); err != nil {
		t.Fatal(err)
	}
}

func TestPrimaryRuntimeNeverSelectsLocalInDelegatedMode(t *testing.T) {
	r, err := NewRuntime(provider.ModeDelegated, nil, nil, WithPrimaryContactManagers(primaryManagerStub{}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.PrimaryContactManager(); CodeOf(err) != CodeDelegateUnavailable {
		t.Fatal("CONTACT_PRIMARY_LOCAL_FALLBACK", err)
	}
}

type primaryManagerStub struct{}

func (primaryManagerStub) SetPrimary(context.Context, SetPrimaryContactInput) (*PrimaryContactResult, error) {
	return nil, nil
}

func TestContactUnicodeValidationAndProtectedCreation(t *testing.T) {
	in := CreateContactInput{CustomerUUID: testCustomerUUID, Status: StatusActive, CreationIntent: CreationIntentExplicitCreate}
	for _, name := range []string{strings.Repeat("界", 128), strings.Repeat("😀", 128)} {
		in.DisplayName = name
		if err := ValidateCreateInput(in); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{strings.Repeat("界", 129), "fixture\x00name", string([]byte{0xff})} {
		in.DisplayName = name
		if CodeOf(ValidateCreateInput(in)) != CodeInvalidArgument {
			t.Fatal("CONTACT_UNICODE_INVALID_ACCEPTED")
		}
	}
	in.DisplayName = "fixture"
	in.Roles = []Role{RolePrimary}
	if CodeOf(ValidateCreateInput(in)) != CodePrimaryOperationRequired {
		t.Fatal("CONTACT_PRIMARY_CREATE_ACCEPTED")
	}
	bad := strings.Repeat("😀", 129)
	if CodeOf(ValidateUpdateInput(UpdateContactInput{CustomerUUID: testCustomerUUID, ContactUUID: testContactUUID, GivenName: &bad})) != CodeInvalidArgument {
		t.Fatal("CONTACT_UNICODE_PATCH_ACCEPTED")
	}
}
