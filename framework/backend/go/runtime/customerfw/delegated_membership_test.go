package customerfw

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnavailableDelegatedMembershipResolverFailsClosed(t *testing.T) {
	resolver := NewUnavailableDelegatedMembershipResolver()
	if _, err := resolver.Resolve(context.Background(), "customer-uuid", "tenant-uuid"); CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("resolve code=%q err=%v", CodeOf(err), err)
	}
	if _, err := resolver.List(context.Background(), "customer-uuid"); CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("list code=%q err=%v", CodeOf(err), err)
	}
}

func TestDelegatedMembershipResolverUsesTwoCredentialsAndNoIdentityParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer plugin-sts" {
			t.Fatalf("unexpected service authorization: %q", got)
		}
		if got := r.Header.Get(customerDelegationHeader); got != "Bearer customer-jwt" {
			t.Fatalf("unexpected customer authorization: %q", got)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("identity must not be serialized as a query: %q", r.URL.RawQuery)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Fatalf("identity must not be serialized as a body: %q", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"item":{"tenant_uuid":"tenant-a","customer_uuid":"customer-a","membership_uuid":"membership-a","status":"active","roles":["customer"],"scopes":["read"]}}}`))
	}))
	defer server.Close()
	resolver, err := NewDelegatedMembershipResolver(DelegatedMembershipResolverConfig{
		BaseURL:       server.URL,
		ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "plugin-sts", nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithCustomerCredential(context.Background(), "customer-jwt")
	got, err := resolver.Resolve(ctx, "customer-a", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.MembershipUUID != "membership-a" {
		t.Fatalf("unexpected membership: %#v", got)
	}
}

func TestDelegatedMembershipResolverMapsCoreErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"CUSTOMER_MEMBERSHIP_INACTIVE"}}`))
	}))
	defer server.Close()
	resolver, err := NewDelegatedMembershipResolver(DelegatedMembershipResolverConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil })})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.List(WithCustomerCredential(context.Background(), "customer"), "customer-a")
	if CodeOf(err) != CodeCustomerMembershipDisabled {
		t.Fatalf("unexpected error: %v", err)
	}
}
