package customerfw

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDelegatedCoreAuthUsesFormalSTSAndCustomerCredentialHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer service-sts" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/tenant/customer/auth/login":
			var in struct {
				Channel    string             `json:"channel"`
				Credential CustomerCredential `json:"credential"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatal(err)
			}
			if in.Channel != ShopifyStorefrontChannel || in.Credential.Value != "storefront-token" {
				t.Fatalf("login input = %#v", in)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"customer-jwt","refresh_token":"refresh","expires_in":60,"context":{"tenant_uuid":"tenant-uuid","customer_uuid":"customer-uuid","membership_uuid":"membership-uuid","status":"active","roles":["customer"],"authenticated":true}}}`))
		case "/api/v1/tenant/customer/auth/validate":
			if got := r.Header.Get(customerDelegationHeader); got != "Bearer customer-jwt" {
				t.Fatalf("customer credential = %q", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"item":{"tenant_uuid":"tenant-uuid","customer_uuid":"customer-uuid","membership_uuid":"membership-uuid","status":"active","roles":["customer"]}}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Login(context.Background(), LoginInput{Channel: ShopifyStorefrontChannel, Credential: CustomerCredential{Type: "shopify_customer_access_token", Value: "storefront-token"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Context.CustomerUUID != "customer-uuid" {
		t.Fatalf("result = %#v", result)
	}
	validated, err := client.Validate(WithCustomerCredential(context.Background(), result.AccessToken), result.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if validated.MembershipUUID != "membership-uuid" {
		t.Fatalf("validated = %#v", validated)
	}
}

func TestDelegatedCoreAuthMapsStableCoreReasons(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"CUSTOMER_IDENTITY_NOT_FOUND"}}`))
	}))
	defer server.Close()
	client, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Login(context.Background(), LoginInput{Channel: ShopifyStorefrontChannel, Credential: CustomerCredential{Type: "shopify_customer_access_token", Value: "storefront-token"}})
	var customerErr *Error
	if !errors.As(err, &customerErr) || customerErr.Code != CodeCustomerIdentityNotFound {
		t.Fatalf("error = %#v", err)
	}
}

func TestDelegatedCoreAuthForwardsWeChatMiniAppCodeWithoutTenant(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/tenant/customer/auth/register" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		credential, _ := body["credential"].(map[string]any)
		if body["channel"] != CustomerAuthChannelWeChatMiniApp || body["app_id"] != "wx-scora" || credential["type"] != WeChatMiniAppCredentialType || credential["value"] != "wx-code" {
			t.Fatalf("body = %#v", body)
		}
		if _, ok := body["tenant_uuid"]; ok {
			t.Fatalf("tenant_uuid must not be forwarded: %#v", body)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"customer-jwt","refresh_token":"refresh","expires_in":60,"context":{"tenant_uuid":"tenant-uuid","customer_uuid":"customer-uuid","membership_uuid":"membership-uuid","status":"active","roles":["customer"],"authenticated":true}}}`))
	}))
	defer server.Close()
	client, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Register(context.Background(), RegisterInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: " wx-scora ", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "wx-code"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Context.CustomerUUID != "customer-uuid" {
		t.Fatalf("result = %#v", result)
	}
	for _, bad := range []RegisterInput{
		{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx-scora", Credential: CustomerCredential{Type: "shopify_customer_access_token", Value: "wx-code"}},
		{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx-scora", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType}},
		{Channel: "unknown", AppID: "wx-scora", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "wx-code"}},
		{Channel: CustomerAuthChannelWeChatMiniApp, Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "wx-code"}},
		{Channel: CustomerAuthChannelDouyinMiniApp, AppID: "tt1", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "code"}},
	} {
		if _, err := client.Register(context.Background(), bad); err == nil {
			t.Fatalf("expected rejection for %#v", bad)
		}
	}
	if calls != 1 {
		t.Fatalf("rejected inputs must not reach Core, calls = %d", calls)
	}
	_, err = client.Login(context.Background(), LoginInput{Channel: CustomerAuthChannelWeChatMiniApp, Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "wx-code"}})
	if CodeOf(err) != CodeCustomerInvalidArgument {
		t.Fatalf("login without app_id = %v", err)
	}
}

func TestDelegatedCoreAuthMapsClientAppErrors(t *testing.T) {
	for reason, want := range map[string]struct {
		code   ErrorCode
		status int
	}{
		"CUSTOMER_CLIENT_APP_NOT_FOUND":        {CodeCustomerClientAppNotFound, http.StatusBadRequest},
		"CUSTOMER_CLIENT_APP_FORBIDDEN":        {CodeCustomerClientAppForbidden, http.StatusForbidden},
		"CUSTOMER_CLIENT_PLATFORM_UNSUPPORTED": {CodeCustomerClientPlatformUnsupported, http.StatusBadRequest},
		"CUSTOMER_CLIENT_APP_NOT_CONFIGURED":   {CodeCustomerClientAppNotConfigured, http.StatusServiceUnavailable},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(want.status)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"` + reason + `","reason_code":"` + reason + `"}}`))
		}))
		client, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), Client: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Register(context.Background(), RegisterInput{Channel: CustomerAuthChannelWeChatMiniApp, AppID: "wx1", Credential: CustomerCredential{Type: WeChatMiniAppCredentialType, Value: "code"}})
		server.Close()
		if CodeOf(err) != want.code || HTTPStatus(err) != want.status || ReasonOf(err) != reason {
			t.Fatalf("%s => code %s status %d reason %s", reason, CodeOf(err), HTTPStatus(err), ReasonOf(err))
		}
	}
}
