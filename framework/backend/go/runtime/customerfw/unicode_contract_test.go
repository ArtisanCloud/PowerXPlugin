package customerfw

import (
	"context"
	"strings"
	"testing"
)

func TestAccountUnicodeValidationBeforeTransport(t *testing.T) {
	stub := &accountSelectorInvoker{}
	client, err := NewAccountSelectorClient(stub)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{strings.Repeat("界", 129), "fixture\x00name", "\tfixture", string([]byte{0xff})} {
		if _, err := client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{DisplayName: name}); err == nil {
			t.Fatal("CUSTOMER_UNICODE_CREATE_ACCEPTED")
		}
		if _, err := client.UpdateBasicAccount(context.Background(), UpdateBasicAccountRequest{CustomerUUID: "11111111-1111-4111-8111-111111111111", DisplayName: &name}); err == nil {
			t.Fatal("CUSTOMER_UNICODE_UPDATE_ACCEPTED")
		}
	}
	if stub.request.CapabilityID != "" {
		t.Fatal("CUSTOMER_UNICODE_TRANSPORT_CALLED")
	}
	name := strings.Repeat("😀", 128)
	if _, err := client.CreateBasicAccount(context.Background(), CreateBasicAccountRequest{DisplayName: name}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdentityCustomer(ExternalIdentityCustomer{Type: "person", DisplayName: name, PrimaryContact: &PrimaryContactInput{DisplayName: name}}); err != nil {
		t.Fatal(err)
	}
	name = strings.Repeat("😀", 129)
	if err := ValidateIdentityCustomer(ExternalIdentityCustomer{Type: "person", DisplayName: "fixture", PrimaryContact: &PrimaryContactInput{DisplayName: name}}); err == nil {
		t.Fatal("CUSTOMER_PRIMARY_UNICODE_ACCEPTED")
	}
}
