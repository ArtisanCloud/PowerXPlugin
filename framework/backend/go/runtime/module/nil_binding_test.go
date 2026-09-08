package module

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

func TestTypedNilAdapterIsUnavailable(t *testing.T) {
	var pointer *int
	var adapter any = pointer
	f, err := NewFactory("test", provider.ModeDelegated, Binding[any]{}, Binding[any]{Value: adapter, Available: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Resolve(); err == nil {
		t.Fatal("typed nil escaped factory")
	}
}
