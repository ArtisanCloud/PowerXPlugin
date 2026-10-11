package customerfw

import (
	"context"
	"testing"
)

func TestUnavailableDelegatedCustomerAuthClientFailsClosed(t *testing.T) {
	client := NewUnavailableDelegatedCustomerAuthClient()
	if _, err := client.Login(context.Background(), LoginInput{}); CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("Login code=%s err=%v", CodeOf(err), err)
	}
}
