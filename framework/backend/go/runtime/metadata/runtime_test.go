package metadata

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

func TestRuntimeSelectsOnlyStartupModeAdapter(t *testing.T) {
	local := &HostClient{}
	delegated := &HostClient{}
	for _, tc := range []struct {
		mode provider.Mode
		want Service
	}{
		{provider.ModeLocal, local}, {provider.ModeDelegated, delegated},
	} {
		runtime, err := NewRuntime(tc.mode, local, delegated)
		if err != nil {
			t.Fatalf("NewRuntime(%s): %v", tc.mode, err)
		}
		got, err := runtime.Service()
		if err != nil {
			t.Fatalf("Service(%s): %v", tc.mode, err)
		}
		if got != tc.want {
			t.Fatalf("Service(%s) selected wrong adapter", tc.mode)
		}
	}
}

func TestRuntimeFailsClosedWhenDelegatedAdapterIsAbsent(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, &HostClient{}, nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if _, err := runtime.Service(); err == nil {
		t.Fatal("expected unavailable delegated adapter")
	}
}
