package miniapp

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestCustomerHandlerUsesRuntimeModeWithoutLocalFallback(t *testing.T) {
	runtime, err := customerfw.NewRuntime(provider.ModeDelegated, customerfw.RuntimeAdapters{}, customerfw.RuntimeAdapters{})
	if err != nil {
		t.Fatal(err)
	}
	h := NewCustomerHandler(&app.Deps{CustomerRuntime: runtime, Config: &config.Config{CustomerAuth: &config.CustomerAuthConfig{Mode: "local"}}})
	if !h.useDelegate() {
		t.Fatal("runtime mode overridden by legacy config")
	}
	if _, err := h.auth.Validate(t.Context(), "token"); err == nil {
		t.Fatal("missing adapter accepted")
	}
	h = NewCustomerHandler(&app.Deps{ProviderMode: provider.ModeDelegated})
	if !h.useDelegate() {
		t.Fatal("delegated mode ignored")
	}
	if _, err := h.auth.Validate(t.Context(), "token"); err == nil {
		t.Fatal("missing runtime accepted")
	}
}
