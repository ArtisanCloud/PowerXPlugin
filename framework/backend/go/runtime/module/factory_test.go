package module

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

func TestFactorySelectsOnlyConfiguredMode(t *testing.T) {
	factory, err := NewFactory("example", provider.ModeDelegated, Binding[string]{Value: "local", Available: true}, Binding[string]{Value: "delegated", Available: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := factory.Resolve()
	if err != nil || got != "delegated" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestFactoryFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	factory, err := NewFactory("example", provider.ModeLocal, Binding[string]{}, Binding[string]{Value: "delegated", Available: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = factory.Resolve()
	if err == nil || err.(*Error).Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("err=%v", err)
	}
}
