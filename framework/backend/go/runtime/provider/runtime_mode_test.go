package provider

import (
	"errors"
	"testing"
)

func TestRuntimeModeMatrix(t *testing.T) {
	for _, mode := range []Mode{ModeLocal, ModeDelegated} {
		for _, proxy := range []string{"0", "1"} {
			t.Run(string(mode)+proxy, func(t *testing.T) {
				got, _, err := ResolveRuntimeMode(ResolveInput{EnvMode: string(mode), PowerXProxy: proxy})
				if err != nil || got.ProviderMode != mode || got.ProxyEnabled != (proxy == "1") {
					t.Fatal("RUNTIME_MODE_MATRIX", got, err)
				}
				want := "redis"
				if mode == ModeDelegated {
					want = "host"
				}
				if got.TaskBusProvider() != want {
					t.Fatal("RUNTIME_TRANSPORT_MATRIX")
				}
			})
		}
	}
}

func TestRuntimeModeRejectsUnknownProxy(t *testing.T) {
	_, _, err := ResolveRuntimeMode(ResolveInput{EnvMode: "local", PowerXProxy: "true"})
	if !errors.Is(err, ErrProxyInvalid) {
		t.Fatal("UNKNOWN_PROXY_ACCEPTED", err)
	}
}
