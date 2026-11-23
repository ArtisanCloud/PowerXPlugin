package customerfw_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"gopkg.in/yaml.v3"
)

const verificationTenant = "22222222-2222-4222-8222-222222222222"
const verificationCustomer = "11111111-1111-4111-8111-111111111111"

func TestServiceMembershipAuthoritativeSchema(t *testing.T) {
	raw, err := os.ReadFile("testdata/customer-membership-service-verification.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile("testdata/customer-membership-service-source.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		SHA256 map[string]string `json:"sha256"`
	}
	if json.Unmarshal(meta, &source) != nil {
		t.Fatal("SERVICE_MEMBERSHIP_SOURCE_INVALID")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != source.SHA256["customer-membership-service-verification.openapi.yaml"] {
		t.Fatal("SERVICE_MEMBERSHIP_SCHEMA_HASH_CHANGED")
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Required             []string `yaml:"required"`
				AdditionalProperties *bool    `yaml:"additionalProperties"`
				Properties           map[string]struct {
					Nullable bool `yaml:"nullable"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"VerifyCustomerRequest", "VerifyCustomerPayload", "VerifiedCustomer"} {
		s := spec.Components.Schemas[name]
		if s.AdditionalProperties == nil || *s.AdditionalProperties {
			t.Fatal("SERVICE_MEMBERSHIP_SCHEMA_NOT_CLOSED")
		}
	}
	if !reflect.DeepEqual(spec.Components.Schemas["VerifyCustomerRequest"].Required, []string{"operation", "schema_version", "customer_uuid"}) {
		t.Fatal("SERVICE_MEMBERSHIP_REQUEST_SCHEMA_CHANGED")
	}
	item := verificationFixture(t)["data"].(map[string]any)["payload"].(map[string]any)["item"].(map[string]any)
	s := spec.Components.Schemas["VerifiedCustomer"]
	if len(item) != len(s.Required) || !s.Properties["expires_at"].Nullable {
		t.Fatal("SERVICE_MEMBERSHIP_FIXTURE_SCHEMA_MISMATCH")
	}
	for _, key := range s.Required {
		if _, ok := item[key]; !ok {
			t.Fatal("SERVICE_MEMBERSHIP_FIXTURE_FIELD_MISSING", key)
		}
	}
}

type verificationTransport func(*http.Request) (*http.Response, error)

func (f verificationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func verificationFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("testdata/customer-membership-service.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func verificationJSON(t *testing.T, body any) []byte {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func verificationResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	return verificationRawResponse(status, verificationJSON(t, body))
}

func verificationRawResponse(status int, raw []byte) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Trace-Id": []string{"http-trace"}, "X-Request-Id": []string{"http-request"}}, Body: io.NopCloser(bytes.NewReader(raw))}
}

func verificationGateway(t *testing.T, scheme string, transport verificationTransport) *gateway.Client {
	t.Helper()
	cfg := gateway.Config{BaseURL: "https://core.fixture", APIPrefix: "/custom/v2", TenantUUID: verificationTenant, AuthScheme: scheme, HTTPClient: &http.Client{Transport: transport}}
	if scheme == "apikey" {
		cfg.APIKey = "fixture-key"
	} else {
		cfg.BearerToken = "fixture-sts"
	}
	g, err := gateway.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = g.Close() })
	return g
}

func verificationClient(t *testing.T, g customerfw.ServiceMembershipGateway) *customerfw.ServiceMembershipVerifier {
	t.Helper()
	v, err := customerfw.NewServiceMembershipVerifier(g)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func requireVerificationError(t *testing.T, result *customerfw.VerifiedCustomerMembership, err error, code string, status int) *customerfw.ServiceMembershipVerificationError {
	t.Helper()
	var typed *customerfw.ServiceMembershipVerificationError
	if result != nil || !errors.As(err, &typed) || typed.Code != code || typed.StatusCode != status || err.Error() != code {
		t.Fatalf("SERVICE_MEMBERSHIP_VALIDATION_ERROR_LOST: %v", err)
	}
	return typed
}

func TestServiceMembershipFixedContractAcrossCredentials(t *testing.T) {
	for _, scheme := range []string{"bearer", "apikey"} {
		t.Run(scheme, func(t *testing.T) {
			calls := 0
			g := verificationGateway(t, scheme, func(r *http.Request) (*http.Response, error) {
				calls++
				auth := "Bearer fixture-sts"
				if scheme == "apikey" {
					auth = "ApiKey fixture-key"
				}
				if r.Method != "POST" || r.URL.Path != "/custom/v2/tenant/invocations" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != auth || r.Header.Get("X-PowerX-Customer-Authorization") != "" {
					t.Error("SERVICE_MEMBERSHIP_TRANSPORT_CHANGED")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"capability_id": customerfw.CapabilityCustomerMembershipsServiceRead, "preferred_protocol": "core_internal", "payload": map[string]any{"body": map[string]any{"operation": "verify_customer", "schema_version": float64(1), "customer_uuid": verificationCustomer}}}
				if !reflect.DeepEqual(body, want) {
					t.Fatal("SERVICE_MEMBERSHIP_REQUEST_OVERRIDE")
				}
				return verificationResponse(t, 200, verificationFixture(t)), nil
			})
			if g.TenantUUID() != verificationTenant {
				t.Fatal("SERVICE_MEMBERSHIP_GATEWAY_TENANT_LOST")
			}
			v := verificationClient(t, g)
			for i := 0; i < 2; i++ {
				ctx := customerfw.WithCustomerCredential(context.Background(), "must-not-be-forwarded")
				result, err := v.Verify(ctx, verificationCustomer)
				if err != nil {
					t.Fatal(err)
				}
				if result.SchemaVersion != 1 || result.CustomerUUID != verificationCustomer || result.TenantUUID != verificationTenant || result.MembershipUUID == "" || result.PrimaryContactUUID == "" || result.Status != "active" || result.ExpiresAt != nil || result.CheckedAt.IsZero() || result.TraceID != "membership-invocation-trace" || result.Revision != "sha256:"+strings.Repeat("a", 64) {
					t.Fatal("SERVICE_MEMBERSHIP_RESULT_INVALID")
				}
			}
			if calls != 2 {
				t.Fatal("SERVICE_MEMBERSHIP_CACHED_OR_RETRIED")
			}
		})
	}
}

func TestServiceMembershipRefreshesServiceCredential(t *testing.T) {
	issued, calls := 0, 0
	g, err := gateway.NewClient(gateway.Config{BaseURL: "https://core.fixture", TenantUUID: verificationTenant, AuthScheme: "bearer", TokenProvider: func(context.Context) (string, error) {
		issued++
		return []string{"fixture-sts-one", "fixture-sts-two"}[issued-1], nil
	}, HTTPClient: &http.Client{Transport: verificationTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != []string{"Bearer fixture-sts-one", "Bearer fixture-sts-two"}[calls-1] {
			t.Error("SERVICE_MEMBERSHIP_STALE_CREDENTIAL")
		}
		return verificationResponse(t, 200, verificationFixture(t)), nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	v := verificationClient(t, g)
	for i := 0; i < 2; i++ {
		if _, err := v.Verify(context.Background(), verificationCustomer); err != nil {
			t.Fatal(err)
		}
	}
	if issued != 2 || calls != 2 {
		t.Fatal("SERVICE_MEMBERSHIP_CREDENTIAL_NOT_REFRESHED")
	}
}

func TestServiceMembershipCoreFailuresAndRevocation(t *testing.T) {
	cases := []struct {
		status int
		reason string
	}{
		{400, "CUSTOMER_VERIFICATION_INVALID_ARGUMENT"}, {401, "CUSTOMER_VERIFICATION_UNAUTHORIZED"},
		{403, "CUSTOMER_VERIFICATION_FORBIDDEN"}, {404, "CUSTOMER_MEMBERSHIP_NOT_FOUND"},
		{409, "CUSTOMER_ACCOUNT_INACTIVE"}, {409, "CUSTOMER_MEMBERSHIP_INACTIVE"},
		{409, "CUSTOMER_MEMBERSHIP_EXPIRED"}, {409, "CUSTOMER_PRIMARY_CONTACT_INVALID"},
		{503, "CUSTOMER_VERIFICATION_UNAVAILABLE"}, {403, "CAPABILITY_INVOKE_FORBIDDEN"},
		{404, "CAPABILITY_NOT_FOUND"},
	}
	for _, scheme := range []string{"bearer", "apikey"} {
		for _, test := range cases {
			t.Run(scheme+"/"+test.reason, func(t *testing.T) {
				calls := 0
				g := verificationGateway(t, scheme, func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return verificationResponse(t, 200, verificationFixture(t)), nil
					}
					return verificationResponse(t, test.status, map[string]any{"success": false, "error_code": "CAPABILITY_INVOCATION_FAILED", "details": map[string]any{"reason_code": test.reason, "trace_id": "business-trace"}}), nil
				})
				v := verificationClient(t, g)
				if _, err := v.Verify(context.Background(), verificationCustomer); err != nil {
					t.Fatal(err)
				}
				result, err := v.Verify(context.Background(), verificationCustomer)
				var typed *gateway.InvocationError
				if result != nil || !errors.As(err, &typed) || typed.StatusCode != test.status || typed.ReasonCode != test.reason || typed.ErrorCode != "CAPABILITY_INVOCATION_FAILED" || typed.TraceID != "http-trace" || typed.RequestID != "http-request" || typed.Details == nil || typed.Details.TraceID != "business-trace" || !bytes.Contains(typed.Body, []byte(test.reason)) {
					t.Fatalf("SERVICE_MEMBERSHIP_CORE_ERROR_REWRITTEN: %v", err)
				}
				if calls != 2 {
					t.Fatal("SERVICE_MEMBERSHIP_REVOCATION_CACHED_OR_RETRIED")
				}
			})
		}
	}
}

func TestServiceMembershipRejectsCorruptResponses(t *testing.T) {
	type mutation struct {
		name   string
		mutate func(map[string]any, map[string]any)
		code   string
		status int
	}
	invalid := customerfw.CodeServiceMembershipResponseInvalid
	cases := []mutation{
		{"version", func(_ map[string]any, i map[string]any) { i["schema_version"] = 2 }, invalid, 502},
		{"version_type", func(_ map[string]any, i map[string]any) { i["schema_version"] = "1" }, invalid, 502},
		{"tenant_mismatch", func(_ map[string]any, i map[string]any) { i["tenant_uuid"] = "55555555-5555-4555-8555-555555555555" }, customerfw.CodeServiceMembershipTenantMismatch, 502},
		{"customer_mismatch", func(_ map[string]any, i map[string]any) { i["customer_uuid"] = "55555555-5555-4555-8555-555555555555" }, customerfw.CodeServiceMembershipCustomerMismatch, 502},
		{"inactive", func(_ map[string]any, i map[string]any) { i["status"] = "inactive" }, invalid, 502},
		{"expiry_type", func(_ map[string]any, i map[string]any) { i["expires_at"] = false }, invalid, 502},
		{"expiry_empty", func(_ map[string]any, i map[string]any) { i["expires_at"] = "" }, invalid, 502},
		{"expiry_offset", func(_ map[string]any, i map[string]any) { i["expires_at"] = "2026-11-12T00:00:00+24:00" }, invalid, 502},
		{"checked_comma_fraction", func(_ map[string]any, i map[string]any) { i["checked_at"] = "2026-11-11T00:00:00,123Z" }, invalid, 502},
		{"expiry_invalid", func(_ map[string]any, i map[string]any) { i["expires_at"] = "invalid" }, invalid, 502},
		{"expired", func(_ map[string]any, i map[string]any) { i["expires_at"] = "2026-11-10T23:59:59Z" }, customerfw.CodeServiceMembershipExpired, 409},
		{"expires_at_checked_at", func(_ map[string]any, i map[string]any) { i["expires_at"] = i["checked_at"] }, customerfw.CodeServiceMembershipExpired, 409},
		{"checked_empty", func(_ map[string]any, i map[string]any) { i["checked_at"] = "" }, invalid, 502},
		{"checked_null", func(_ map[string]any, i map[string]any) { i["checked_at"] = nil }, invalid, 502},
		{"checked_zero", func(_ map[string]any, i map[string]any) { i["checked_at"] = "0001-01-01T00:00:00Z" }, invalid, 502},
		{"checked_non_utc", func(_ map[string]any, i map[string]any) { i["checked_at"] = "2026-11-11T00:00:00+01:00" }, invalid, 502},
		{"revision_type", func(_ map[string]any, i map[string]any) { i["revision"] = 1 }, invalid, 502},
		{"revision_invalid", func(_ map[string]any, i map[string]any) { i["revision"] = "sha256:" + strings.Repeat("A", 64) }, invalid, 502},
		{"unknown_item_field", func(_ map[string]any, i map[string]any) { i["actor"] = "forged" }, invalid, 502},
		{"unknown_payload_field", func(d map[string]any, _ map[string]any) { d["payload"].(map[string]any)["items"] = []any{} }, invalid, 502},
		{"empty_item", func(d map[string]any, _ map[string]any) { d["payload"].(map[string]any)["item"] = nil }, invalid, 502},
		{"payload_type", func(d map[string]any, _ map[string]any) { d["payload"] = []any{} }, invalid, 502},
		{"trace_missing", func(d map[string]any, _ map[string]any) { delete(d, "trace_id") }, invalid, 502},
		{"trace_empty", func(d map[string]any, _ map[string]any) { d["trace_id"] = "" }, invalid, 502},
		{"trace_control", func(d map[string]any, _ map[string]any) { d["trace_id"] = "trace\nvalue" }, invalid, 502},
	}
	for _, field := range []string{"schema_version", "tenant_uuid", "customer_uuid", "membership_uuid", "primary_contact_uuid", "status", "expires_at", "checked_at", "revision"} {
		key := field
		cases = append(cases, mutation{"missing_" + key, func(_ map[string]any, i map[string]any) { delete(i, key) }, invalid, 502})
		cases = append(cases, mutation{"case_" + key, func(_ map[string]any, i map[string]any) { i[strings.ToUpper(key)] = i[key]; delete(i, key) }, invalid, 502})
	}
	for _, field := range []string{"tenant_uuid", "customer_uuid", "membership_uuid", "primary_contact_uuid"} {
		for _, value := range []string{"", "00000000-0000-0000-0000-000000000000", "ABCDEFAB-1234-4234-8234-ABCDEFABCDEF", "11111111111141118111111111111111", " 11111111-1111-4111-8111-111111111111", "42"} {
			key, raw := field, value
			cases = append(cases, mutation{key + "_" + raw, func(_ map[string]any, i map[string]any) { i[key] = raw }, invalid, 502})
		}
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			body := verificationFixture(t)
			data := body["data"].(map[string]any)
			item := data["payload"].(map[string]any)["item"].(map[string]any)
			test.mutate(data, item)
			g := verificationGateway(t, "apikey", func(*http.Request) (*http.Response, error) { return verificationResponse(t, 200, body), nil })
			result, err := verificationClient(t, g).Verify(context.Background(), verificationCustomer)
			typed := requireVerificationError(t, result, err, test.code, test.status)
			if !strings.HasPrefix(test.name, "trace_") && typed.TraceID != "membership-invocation-trace" {
				t.Fatal("SERVICE_MEMBERSHIP_VALIDATION_TRACE_LOST")
			}
		})
	}
}

func TestServiceMembershipRejectsDuplicateWireFields(t *testing.T) {
	for _, key := range []string{"schema_version", "expires_at", "item", "payload", "trace_id"} {
		t.Run(key, func(t *testing.T) {
			raw := string(verificationJSON(t, verificationFixture(t)))
			raw = strings.Replace(raw, `"`+key+`":`, `"`+key+`":null,"`+key+`":`, 1)
			g := verificationGateway(t, "apikey", func(*http.Request) (*http.Response, error) { return verificationRawResponse(200, []byte(raw)), nil })
			result, err := verificationClient(t, g).Verify(context.Background(), verificationCustomer)
			requireVerificationError(t, result, err, customerfw.CodeServiceMembershipResponseInvalid, 502)
		})
	}
}

type verificationStub struct {
	tenant   string
	response *gateway.Response
	err      error
	calls    int
}

func (s *verificationStub) TenantUUID() string { return s.tenant }
func (s *verificationStub) Invoke(context.Context, gateway.InvokeRequest) (*gateway.Response, error) {
	s.calls++
	return s.response, s.err
}

func TestServiceMembershipConstructionAndInput(t *testing.T) {
	if _, err := customerfw.NewServiceMembershipVerifier(nil); err == nil {
		t.Fatal("SERVICE_MEMBERSHIP_NIL_GATEWAY_ACCEPTED")
	}
	var nilGateway *gateway.Client
	if nilGateway.TenantUUID() != "" {
		t.Fatal("SERVICE_MEMBERSHIP_NIL_TENANT")
	}
	for _, tenant := range []string{"", "00000000-0000-0000-0000-000000000000", "ABCDEFAB-1234-4234-8234-ABCDEFABCDEF", "tenant"} {
		if _, err := customerfw.NewServiceMembershipVerifier(&verificationStub{tenant: tenant}); err == nil {
			t.Fatal("SERVICE_MEMBERSHIP_INVALID_TENANT_ACCEPTED")
		}
	}
	s := &verificationStub{tenant: verificationTenant}
	v := verificationClient(t, s)
	for _, customer := range []string{"", "00000000-0000-0000-0000-000000000000", "ABCDEFAB-1234-4234-8234-ABCDEFABCDEF", "11111111111141118111111111111111", "42"} {
		result, err := v.Verify(context.Background(), customer)
		requireVerificationError(t, result, err, customerfw.CodeServiceMembershipInvalidArgument, 400)
	}
	result, err := v.Verify(nil, verificationCustomer)
	requireVerificationError(t, result, err, customerfw.CodeServiceMembershipInvalidArgument, 400)
	s.tenant = "55555555-5555-4555-8555-555555555555"
	result, err = v.Verify(context.Background(), verificationCustomer)
	requireVerificationError(t, result, err, customerfw.CodeServiceMembershipTenantMismatch, 502)
	if s.calls != 0 {
		t.Fatal("SERVICE_MEMBERSHIP_INVALID_INPUT_INVOKED")
	}
	var nilVerifier *customerfw.ServiceMembershipVerifier
	result, err = nilVerifier.Verify(context.Background(), verificationCustomer)
	requireVerificationError(t, result, err, customerfw.CodeServiceMembershipGatewayRequired, 503)
}

func TestServiceMembershipRequiresRawGatewayDataAndPreservesErrors(t *testing.T) {
	for _, response := range []*gateway.Response{nil, {}, {Data: verificationFixture(t)["data"].(map[string]any)}} {
		s := &verificationStub{tenant: verificationTenant, response: response}
		result, err := verificationClient(t, s).Verify(context.Background(), verificationCustomer)
		requireVerificationError(t, result, err, customerfw.CodeServiceMembershipResponseInvalid, 502)
	}
	want := &gateway.InvocationError{StatusCode: 503, ErrorCode: "CAPABILITY_UNAVAILABLE", ReasonCode: "CUSTOMER_VERIFICATION_UNAVAILABLE", TraceID: "core-trace"}
	s := &verificationStub{tenant: verificationTenant, err: want}
	result, err := verificationClient(t, s).Verify(context.Background(), verificationCustomer)
	if result != nil || err != want || s.calls != 1 {
		t.Fatal("SERVICE_MEMBERSHIP_ORIGINAL_ERROR_LOST")
	}
}

func TestServiceMembershipMalformedRawData(t *testing.T) {
	base := verificationJSON(t, verificationFixture(t)["data"])
	invalidUTF8 := bytes.Replace(base, []byte("membership-invocation-trace"), []byte{0xff}, 1)
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`{`), append(append([]byte{}, base...), []byte(` {}`)...), invalidUTF8} {
		s := &verificationStub{tenant: verificationTenant, response: &gateway.Response{RawData: raw}}
		result, err := verificationClient(t, s).Verify(context.Background(), verificationCustomer)
		requireVerificationError(t, result, err, customerfw.CodeServiceMembershipResponseInvalid, 502)
	}
}

func TestServiceMembershipValidExpiry(t *testing.T) {
	body := verificationFixture(t)
	body["data"].(map[string]any)["payload"].(map[string]any)["item"].(map[string]any)["expires_at"] = "2026-11-12T00:00:00Z"
	g := verificationGateway(t, "apikey", func(*http.Request) (*http.Response, error) { return verificationResponse(t, 200, body), nil })
	result, err := verificationClient(t, g).Verify(context.Background(), verificationCustomer)
	if err != nil || result == nil || result.ExpiresAt == nil || !result.ExpiresAt.After(result.CheckedAt) {
		t.Fatal("SERVICE_MEMBERSHIP_VALID_EXPIRY_REJECTED", err)
	}
}

func TestServiceMembershipCancellationAndDeadline(t *testing.T) {
	t.Run("pre_canceled", func(t *testing.T) {
		s := &verificationStub{tenant: verificationTenant}
		v := verificationClient(t, s)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := v.Verify(ctx, verificationCustomer)
		if result != nil || !errors.Is(err, context.Canceled) || s.calls != 0 {
			t.Fatal("SERVICE_MEMBERSHIP_CANCELED_INPUT_INVOKED", err)
		}
	})
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			started := make(chan struct{})
			g := verificationGateway(t, "bearer", func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				close(started)
				<-r.Context().Done()
				return nil, r.Context().Err()
			})
			v := verificationClient(t, g)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if mode == "deadline" {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 20*time.Millisecond)
				defer stop()
			}
			done := make(chan error, 1)
			go func() { _, err := v.Verify(ctx, verificationCustomer); done <- err }()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("SERVICE_MEMBERSHIP_TRANSPORT_NOT_STARTED")
			}
			want := context.DeadlineExceeded
			if mode == "cancel" {
				cancel()
				want = context.Canceled
			}
			select {
			case err := <-done:
				if !errors.Is(err, want) {
					t.Fatal("SERVICE_MEMBERSHIP_CONTEXT_LOST", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("SERVICE_MEMBERSHIP_CONTEXT_IGNORED")
			}
			if calls.Load() != 1 {
				t.Fatal("SERVICE_MEMBERSHIP_CANCELLATION_RETRIED")
			}
		})
	}
}
