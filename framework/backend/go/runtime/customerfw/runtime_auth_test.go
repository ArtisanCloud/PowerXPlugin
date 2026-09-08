package customerfw

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

func TestRuntimeAuthClientNeverFallsBack(t *testing.T) {
	r, err := NewRuntime(provider.ModeDelegated, AdaptersFromLocalStore(poisonedCustomerAdapter{}), RuntimeAdapters{})
	if err != nil {
		t.Fatal(err)
	}
	for _, runtime := range []*Runtime{nil, r} {
		client := runtime.AuthClient()
		if _, err := client.Register(t.Context(), RegisterInput{}); err == nil {
			t.Fatal("Register accepted missing adapter")
		}
		if _, err := client.Login(t.Context(), LoginInput{}); err == nil {
			t.Fatal("Login accepted missing adapter")
		}
		if _, err := client.Validate(t.Context(), "token"); CodeOf(err) != CodeCustomerDelegateUnavailable {
			t.Fatal("Validate accepted missing adapter")
		}
	}
}
