package knowledge

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

// Embed the contract only to test selection: no fake persistence is shipped.
type selectionProvider struct{ KnowledgeProvider }

func TestRuntimeSelectsOnlyConfiguredMode(t *testing.T) {
	local, delegated := &selectionProvider{}, &selectionProvider{}
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		runtime, err := NewRuntime(mode, local, delegated)
		if err != nil {
			t.Fatal(err)
		}
		got, err := runtime.Provider()
		if err != nil {
			t.Fatal(err)
		}
		if mode == provider.ModeLocal && got != local || mode == provider.ModeDelegated && got != delegated {
			t.Fatal("provider_selection")
		}
	}
	runtime, _ := NewRuntime(provider.ModeDelegated, local, nil)
	if _, err := runtime.Provider(); err == nil {
		t.Fatal("delegated_fallback")
	}
	var missing *selectionProvider
	runtime, _ = NewRuntime(provider.ModeLocal, missing, delegated)
	if _, err := runtime.Provider(); err == nil {
		t.Fatal("typed_nil")
	}
}
