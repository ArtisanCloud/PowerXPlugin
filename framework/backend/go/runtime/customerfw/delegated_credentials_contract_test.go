package customerfw

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestSharedMembershipClientKeepsRequestCredentialsIsolated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get(customerDelegationHeader), "Bearer jwt-")
		if r.Header.Get("Authorization") != "Bearer plugin-sts" {
			t.Error("service_credential_mismatch")
		}
		if r.URL.RawQuery != "" || r.Header.Get("X-Tenant-UUID") != "" {
			t.Error("identity_override")
		}
		// A standard Core data envelope does not require a success field.
		fmt.Fprintf(w, `{"data":{"item":{"tenant_uuid":"11111111-1111-4111-8111-111111111111","customer_uuid":%q,"membership_uuid":%q,"status":"active"}}}`, token, token)
	}))
	defer server.Close()
	c, err := NewDelegatedMembershipResolver(DelegatedMembershipResolverConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "plugin-sts", nil })})
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			customer := fmt.Sprintf("00000000-0000-4000-8000-%012d", index+1)
			out, err := c.Resolve(WithCustomerCredential(context.Background(), "jwt-"+customer), customer, "11111111-1111-4111-8111-111111111111")
			if err != nil {
				t.Error(err)
				return
			}
			if out.CustomerUUID != customer || out.MembershipUUID != customer {
				t.Errorf("subject_mismatch=%#v", out)
			}
		}(i)
	}
	workers.Wait()
}

func TestFormalCustomerAuthRejectsRawIdentityAndFalseSuccess(t *testing.T) {
	item := `{"tenant_uuid":"11111111-1111-4111-8111-111111111111","customer_uuid":"22222222-2222-4222-8222-222222222222","membership_uuid":"33333333-3333-4333-8333-333333333333","status":"active"}`
	for _, body := range []string{`{"item":` + item + `}`, `{"success":false,"data":{"item":` + item + `}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil })})
		if err != nil {
			t.Fatal(err)
		}
		out, err := c.Validate(WithCustomerCredential(context.Background(), "customer-token"), "")
		server.Close()
		if out != nil || CodeOf(err) != CodeCustomerDelegateUnavailable {
			t.Fatalf("out=%#v err=%v", out, err)
		}
	}
}
