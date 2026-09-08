package customerfw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateRejectsIncompleteOrInactiveMembership(t *testing.T) {
	for _, body := range []string{"{}", "{\"data\":null}", "{\"data\":{\"item\":{}}}", "{\"data\":{\"item\":{\"tenant_uuid\":\"t\",\"customer_uuid\":\"c\",\"membership_uuid\":\"m\",\"status\":\"disabled\"}}}"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
			defer server.Close()
			c, err := NewDelegatedCoreAuthClient(DelegatedCoreAuthClientConfig{BaseURL: server.URL, ServiceTokens: ServiceTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil })})
			if err != nil {
				t.Fatal(err)
			}
			out, err := c.Validate(WithCustomerCredential(context.Background(), "customer-token"), "")
			var typed *Error
			if out != nil || !errors.As(err, &typed) {
				t.Fatalf("out=%#v err=%#v", out, err)
			}
		})
	}
}

func TestCustomerHostStableErrorMatrix(t *testing.T) {
	for _, tc := range []struct {
		status int
		reason string
		code   ErrorCode
	}{
		{400, "CUSTOMER_INVALID_ARGUMENT", CodeCustomerInvalidArgument},
		{401, "CUSTOMER_UNAUTHORIZED", CodeCustomerTokenInvalid},
		{403, "CUSTOMER_FORBIDDEN", CodeCustomerForbidden},
		{404, "CUSTOMER_MEMBERSHIP_NOT_FOUND", CodeCustomerMembershipRequired},
		{403, "CUSTOMER_MEMBERSHIP_INACTIVE", CodeCustomerMembershipDisabled},
		{502, "CUSTOMER_UPSTREAM_DEPENDENCY", CodeCustomerDelegateUnavailable},
		{503, "CUSTOMER_UPSTREAM_DEPENDENCY", CodeCustomerDelegateUnavailable},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			err := mapDelegatedMembershipError(tc.status, []byte(fmt.Sprintf("{\"reason_code\":%q}", tc.reason)))
			var typed *Error
			if !errors.As(err, &typed) || typed.Code != tc.code {
				t.Fatalf("error=%#v", err)
			}
			wrapped := fmt.Errorf("adapter: %w", err)
			if HTTPStatus(wrapped) != tc.status || ReasonOf(wrapped) != tc.reason {
				t.Fatalf("status=%d reason=%s", HTTPStatus(wrapped), ReasonOf(wrapped))
			}
			if HTTPStatus(mapValidatorError(wrapped)) != tc.status || ReasonOf(mapValidatorError(wrapped)) != tc.reason {
				t.Fatal("validator mapping discarded Core error metadata")
			}
		})
	}
}
