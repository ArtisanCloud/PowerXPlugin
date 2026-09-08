package customerfw

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// ExternalIdentityService is shared by local and delegated customer adapters.
type ExternalIdentityService interface {
	ResolveExternalIdentity(context.Context, ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error)
}

// LocalCustomerStore is implemented by a plugin when it owns local customer
// persistence. It deliberately exposes only Framework contracts.
type LocalCustomerStore interface {
	CustomerAuthClient
	ExternalIdentityService
	CustomerMembershipResolver
}

type Runtime struct {
	mode       provider.Mode
	auth       *module.Factory[CustomerAuthClient]
	external   *module.Factory[ExternalIdentityService]
	membership *module.Factory[CustomerMembershipResolver]
}

type RuntimeAdapters struct {
	Auth       CustomerAuthClient
	External   ExternalIdentityService
	Membership CustomerMembershipResolver
}

// AdaptersFromLocalStore exposes a plugin-supplied local store through the
// same three contracts used by the delegated runtime. The Framework owns mode
// selection; the plugin only supplies its local persistence implementation.
func AdaptersFromLocalStore(store LocalCustomerStore) RuntimeAdapters {
	if store == nil {
		return RuntimeAdapters{}
	}
	return RuntimeAdapters{
		Auth:       store,
		External:   store,
		Membership: store,
	}
}

func NewRuntime(mode provider.Mode, local RuntimeAdapters, delegated RuntimeAdapters) (*Runtime, error) {
	auth, err := module.NewFactory("customer.auth", mode,
		module.Binding[CustomerAuthClient]{Value: local.Auth, Available: local.Auth != nil},
		module.Binding[CustomerAuthClient]{Value: delegated.Auth, Available: delegated.Auth != nil},
	)
	if err != nil {
		return nil, err
	}
	external, err := module.NewFactory("customer.external_identity", mode,
		module.Binding[ExternalIdentityService]{Value: local.External, Available: local.External != nil},
		module.Binding[ExternalIdentityService]{Value: delegated.External, Available: delegated.External != nil},
	)
	if err != nil {
		return nil, err
	}
	membership, err := module.NewFactory("customer.membership", mode,
		module.Binding[CustomerMembershipResolver]{Value: local.Membership, Available: local.Membership != nil},
		module.Binding[CustomerMembershipResolver]{Value: delegated.Membership, Available: delegated.Membership != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, auth: auth, external: external, membership: membership}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}
func (r *Runtime) Auth() (CustomerAuthClient, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "customer runtime is unavailable")
	}
	return r.auth.Resolve()
}
func (r *Runtime) ExternalIdentity() (ExternalIdentityService, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "customer runtime is unavailable")
	}
	return r.external.Resolve()
}
func (r *Runtime) Membership() (CustomerMembershipResolver, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "customer runtime is unavailable")
	}
	return r.membership.Resolve()
}
