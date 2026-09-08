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
