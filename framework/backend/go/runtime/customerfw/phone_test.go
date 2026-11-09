package customerfw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

func phoneTestClient(t *testing.T, handler http.HandlerFunc) *DelegatedCoreAuthClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestVerifyPhoneForwardsCodeWithServiceAndCustomerCredentials(t *testing.T) {
	calls := 0
	client := phoneTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/tenant/customer/phone:verify" || r.Header.Get("Authorization") != "Bearer service-sts" || r.Header.Get("X-PowerX-Customer-Authorization") != "Bearer customer-jwt" {
			t.Fatalf("path=%s auth=%q customer=%q", r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("X-PowerX-Customer-Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		credential, _ := body["credential"].(map[string]any)
		if body["channel"] != CustomerAuthChannelWeChatMiniApp || body["app_id"] != "wx-scora" || credential["type"] != WeChatMiniAppPhoneCredentialType || credential["value"] != "phone-code" {
			t.Fatalf("body = %#v", body)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"phone_masked":"138****1234","verified_at":"2026-11-09T12:00:00Z","phone_verified":true}}`))
	})
	ctx := WithCustomerCredential(context.Background(), "customer-jwt")
	out, err := client.VerifyPhone(ctx, VerifyPhoneInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: " wx-scora ", Credential: CustomerCredential{Type: WeChatMiniAppPhoneCredentialType, Value: "phone-code"}})
	if err != nil || out.PhoneMasked != "138****1234" || out.VerifiedAt.IsZero() {
		t.Fatalf("out = %#v err = %v", out, err)
	}

	for _, bad := range []VerifyPhoneInput{
		{Channel: CustomerAuthChannelWeChatMiniApp, Credential: CustomerCredential{Type: WeChatMiniAppPhoneCredentialType, Value: "c"}},
		{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "c"}},
		{Channel: "shopify_storefront", AppID: "wx", Credential: CustomerCredential{Type: WeChatMiniAppPhoneCredentialType, Value: "c"}},
	} {
		if _, err := client.VerifyPhone(ctx, bad); CodeOf(err) != CodeCustomerInvalidArgument {
			t.Fatalf("%#v => %v", bad, err)
		}
	}
	if _, err := client.VerifyPhone(context.Background(), VerifyPhoneInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx", Credential: CustomerCredential{Type: WeChatMiniAppPhoneCredentialType, Value: "c"}}); CodeOf(err) != CodeCustomerTokenMissing {
		t.Fatalf("missing customer credential => %v", err)
	}
	if calls != 1 {
		t.Fatalf("rejected inputs must not reach Core, calls = %d", calls)
	}
}

func TestVerifyPhoneMapsCoreErrors(t *testing.T) {
	for reason, want := range map[string]struct {
		code   ErrorCode
		status int
	}{
		"CUSTOMER_PHONE_CONFLICT":       {CodeCustomerPhoneConflict, http.StatusConflict},
		"CUSTOMER_PHONE_CODE_INVALID":   {CodeCustomerPhoneCodeInvalid, http.StatusUnauthorized},
		"CUSTOMER_CLIENT_APP_FORBIDDEN": {CodeCustomerClientAppForbidden, http.StatusForbidden},
	} {
		client := phoneTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(want.status)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"` + reason + `","reason_code":"` + reason + `"}}`))
		})
		ctx := WithCustomerCredential(context.Background(), "customer-jwt")
		_, err := client.VerifyPhone(ctx, VerifyPhoneInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx", Credential: CustomerCredential{Type: WeChatMiniAppPhoneCredentialType, Value: "c"}})
		if CodeOf(err) != want.code || HTTPStatus(err) != want.status || ReasonOf(err) != reason {
			t.Fatalf("%s => code %s status %d reason %s", reason, CodeOf(err), HTTPStatus(err), ReasonOf(err))
		}
	}
}

func TestLoginResultCarriesPhoneVerified(t *testing.T) {
	client := phoneTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"jwt","expires_in":60,"phone_verified":true,"context":{"tenant_uuid":"t","customer_uuid":"c","membership_uuid":"m","status":"active","authenticated":true}}}`))
	})
	out, err := client.Register(context.Background(), RegisterInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "code"}})
	if err != nil || !out.PhoneVerified {
		t.Fatalf("out = %#v err = %v", out, err)
	}
}

func TestRuntimePhoneVerifierOnlyInDelegatedMode(t *testing.T) {
	store := runtimeStore{}
	local, err := NewRuntime(provider.ModeLocal, AdaptersFromLocalStore(store), RuntimeAdapters{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := local.PhoneVerifier(); CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("local mode => %v", err)
	}
	delegatedClient := phoneTestClient(t, func(http.ResponseWriter, *http.Request) {})
	delegated, err := NewRuntime(provider.ModeDelegated, RuntimeAdapters{}, RuntimeAdapters{Auth: delegatedClient, External: store, Membership: store})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := delegated.PhoneVerifier(); err != nil || v == nil {
		t.Fatalf("delegated mode => %v", err)
	}
}
