package contactfw

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Store is the complete, typed Contact master-data contract. Every operation
// is customer-scoped so implementations can enforce tenant/customer/contact
// ownership together.
type Store interface {
	Create(ctx context.Context, input CreateContactInput) (*Contact, error)
	Get(ctx context.Context, input GetContactInput) (*Contact, error)
	Update(ctx context.Context, input UpdateContactInput) (*Contact, error)
	ListByCustomer(ctx context.Context, input ListByCustomerInput) (ContactPage, error)
	ResolveIdentity(ctx context.Context, input ResolveContactIdentityInput) (*ContactIdentityResolution, error)
	BindIdentity(ctx context.Context, input BindContactIdentityInput) (*ContactIdentity, error)
}

// LocalStore is implemented by a consuming plugin only in local mode.
type LocalStore interface{ Store }

// IdentityChannelMigrator is intentionally separate from Store: it is an
// explicit local data-repair capability, not an ordinary business operation
// that a delegated Core client may emulate.
type IdentityChannelMigrator interface {
	MigrateIdentityChannel(ctx context.Context, input MigrateContactIdentityChannelInput) (*ContactIdentity, error)
}

// DelegatedContactClient is a typed Core internal binding. It intentionally
// exposes no method/endpoint/body escape hatch and no raw Bearer forwarding.
type DelegatedContactClient interface{ Store }

type Runtime struct {
	mode  provider.Mode
	store *module.Factory[Store]
}

func NewRuntime(mode provider.Mode, local LocalStore, delegated DelegatedContactClient) (*Runtime, error) {
	store, err := module.NewFactory("contact.store", mode,
		module.Binding[Store]{Value: local, Available: local != nil},
		module.Binding[Store]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, store: store}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

// Store returns only the adapter bound to the trusted startup mode.
func (r *Runtime) Store() (Store, error) {
	if r == nil || r.store == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "contact runtime is unavailable")
	}
	return r.store.Resolve()
}

// ValidateRequired is called by a plugin's bootstrap when it declares or
// consumes Contact. It turns a selected-adapter omission into startup failure.
func (r *Runtime) ValidateRequired() error {
	if _, err := r.Store(); err != nil {
		if r != nil && r.Mode() == provider.ModeDelegated {
			return NewError(CodeDelegateUnavailable, err)
		}
		return NewError(CodeLocalUnavailable, err)
	}
	return nil
}
