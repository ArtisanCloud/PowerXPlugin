package adapters

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"testing"
)

func TestRegistryRejectsTypedNilWithoutCommittingBinding(t *testing.T) {
	var directory *stubDirectory
	var authz *stubAuthz
	var identity *stubIdentityContext
	for _, bundle := range []Bundle{
		{Directory: directory, Authz: stubAuthz{}, Context: stubIdentityContext{}},
		{Directory: stubDirectory{}, Authz: authz, Context: stubIdentityContext{}},
		{Directory: stubDirectory{}, Authz: stubAuthz{}, Context: identity},
	} {
		r := NewRegistry()
		if err := r.Bind(contracts.IAMAdapterModeLocal, bundle); err == nil {
			t.Fatal("nil_adapter_bound")
		}
		if r.IsBound() {
			t.Fatal("invalid_binding_committed")
		}
		if err := r.Bind(contracts.IAMAdapterModeLocal, Bundle{Directory: stubDirectory{}, Authz: stubAuthz{}, Context: stubIdentityContext{}}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestNilRegistryFailsClosed(t *testing.T) {
	var r *Registry
	if r.IsBound() {
		t.Fatal("nil_registry_bound")
	}
	if _, ok := r.Mode(); ok {
		t.Fatal("nil_registry_mode")
	}
	if _, err := r.Directory(); err == nil {
		t.Fatal("nil_directory")
	}
	if _, err := r.Authz(); err == nil {
		t.Fatal("nil_authz")
	}
	if _, err := r.IdentityContext(); err == nil {
		t.Fatal("nil_context")
	}
	if err := r.Bind(contracts.IAMAdapterModeLocal, Bundle{}); err == nil {
		t.Fatal("nil_bind")
	}
}
