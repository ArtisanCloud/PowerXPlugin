package customerfw

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

type runtimeStore struct{}

func (runtimeStore) Register(context.Context, RegisterInput) (*AuthResult, error) { return nil, nil }
func (runtimeStore) Login(context.Context, LoginInput) (*AuthResult, error)       { return nil, nil }
func (runtimeStore) Validate(context.Context, string) (*CustomerContext, error)   { return nil, nil }
func (runtimeStore) ResolveExternalIdentity(context.Context, ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error) {
	return nil, nil
}
func (runtimeStore) List(context.Context, string) ([]CustomerMembership, error) { return nil, nil }
func (runtimeStore) Resolve(context.Context, string, string) (*CustomerMembership, error) {
	return nil, nil
}
func TestRuntimeSelectsLocalAdapters(t *testing.T) {
	store := runtimeStore{}
	r, err := NewRuntime(provider.ModeLocal, AdaptersFromLocalStore(store), RuntimeAdapters{})
	auth, authErr := r.Auth()
	if err != nil || authErr != nil || auth == nil || r.Mode() != provider.ModeLocal {
		t.Fatalf("runtime=%#v err=%v authErr=%v", r, err, authErr)
	}
}

func TestRuntimeFailsOnlyForTheUnavailableDelegatedOperation(t *testing.T) {
	store := runtimeStore{}
	r, err := NewRuntime(provider.ModeDelegated, RuntimeAdapters{Auth: store, External: store, Membership: store}, RuntimeAdapters{
		Auth:       store,
		Membership: NewUnavailableDelegatedMembershipResolver(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Auth(); err != nil {
		t.Fatalf("auth should be available: %v", err)
	}
	membership, err := r.Membership()
	if err != nil {
		t.Fatalf("membership adapter must be explicitly installed: %v", err)
	}
	if _, err := membership.Resolve(context.Background(), "customer-uuid", "tenant-uuid"); CodeOf(err) != CodeCustomerDelegateUnavailable {
		t.Fatalf("membership must fail closed: code=%q err=%v", CodeOf(err), err)
	}
}

func TestRuntimeLocalAndDelegatedAdaptersReturnEquivalentCustomerContractShape(t *testing.T) {
	local := contractRuntimeAdapter{source: CustomerAuthSourceLocal}
	delegated := contractRuntimeAdapter{source: CustomerAuthSourceDelegate}

	localRuntime, err := NewRuntime(provider.ModeLocal, AdaptersFromLocalStore(local), RuntimeAdapters{})
	if err != nil {
		t.Fatalf("NewRuntime(local): %v", err)
	}
	delegatedRuntime, err := NewRuntime(provider.ModeDelegated, RuntimeAdapters{Auth: poisonedCustomerAdapter{}, External: poisonedCustomerAdapter{}, Membership: poisonedCustomerAdapter{}}, RuntimeAdapters{Auth: delegated, External: delegated, Membership: delegated})
	if err != nil {
		t.Fatalf("NewRuntime(delegated): %v", err)
	}

	for _, tc := range []struct {
		name    string
		runtime *Runtime
	}{
		{name: "local", runtime: localRuntime},
		{name: "delegated", runtime: delegatedRuntime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth, err := tc.runtime.Auth()
			if err != nil {
				t.Fatalf("Auth(): %v", err)
			}
			customerContext, err := auth.Validate(context.Background(), "customer-token")
			if err != nil {
				t.Fatalf("Validate(): %v", err)
			}
			assertCustomerContextShape(t, customerContext)

			external, err := tc.runtime.ExternalIdentity()
			if err != nil {
				t.Fatalf("ExternalIdentity(): %v", err)
			}
			identity, err := external.ResolveExternalIdentity(context.Background(), ResolveExternalIdentityRequest{ProviderSubject: "subject-1", DisplayName: "Customer"})
			if err != nil || identity == nil || identity.CustomerUUID == "" || identity.MembershipUUID == "" || identity.DisplayName == "" {
				t.Fatalf("ResolveExternalIdentity() = %#v, %v", identity, err)
			}

			membership, err := tc.runtime.Membership()
			if err != nil {
				t.Fatalf("Membership(): %v", err)
			}
			resolved, err := membership.Resolve(context.Background(), customerContext.CustomerUUID, customerContext.TenantUUID)
			if err != nil || resolved == nil || resolved.CustomerUUID != customerContext.CustomerUUID || resolved.TenantUUID != customerContext.TenantUUID || len(resolved.Roles) == 0 {
				t.Fatalf("Membership.Resolve() = %#v, %v", resolved, err)
			}
		})
	}
}

func assertCustomerContextShape(t *testing.T, value *CustomerContext) {
	t.Helper()
	if value == nil || value.TenantUUID == "" || value.CustomerUUID == "" || value.MembershipUUID == "" || !value.Authenticated || value.Profile.DisplayName == "" || len(value.Roles) == 0 || value.Source == "" {
		t.Fatalf("invalid CustomerContext shape: %#v", value)
	}
}

type contractRuntimeAdapter struct{ source CustomerAuthSource }

func (a contractRuntimeAdapter) Register(context.Context, RegisterInput) (*AuthResult, error) {
	return nil, nil
}
func (a contractRuntimeAdapter) Login(context.Context, LoginInput) (*AuthResult, error) {
	return nil, nil
}
func (a contractRuntimeAdapter) Validate(context.Context, string) (*CustomerContext, error) {
	return &CustomerContext{TenantUUID: "tenant-uuid", CustomerUUID: "customer-uuid", MembershipUUID: "membership-uuid", Profile: CustomerAttributes{DisplayName: "Customer"}, Roles: []string{"customer"}, Source: a.source, Authenticated: true}, nil
}
func (a contractRuntimeAdapter) ResolveExternalIdentity(context.Context, ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error) {
	return &ExternalIdentityResolution{CustomerUUID: "customer-uuid", MembershipUUID: "membership-uuid", DisplayName: "Customer"}, nil
}
func (a contractRuntimeAdapter) List(context.Context, string) ([]CustomerMembership, error) {
	return []CustomerMembership{{CustomerUUID: "customer-uuid", TenantUUID: "tenant-uuid", MembershipUUID: "membership-uuid", Roles: []string{"customer"}, Status: CustomerMembershipActive}}, nil
}
func (a contractRuntimeAdapter) Resolve(context.Context, string, string) (*CustomerMembership, error) {
	return &CustomerMembership{CustomerUUID: "customer-uuid", TenantUUID: "tenant-uuid", MembershipUUID: "membership-uuid", Roles: []string{"customer"}, Status: CustomerMembershipActive}, nil
}

type poisonedCustomerAdapter struct{}

func (poisonedCustomerAdapter) Register(context.Context, RegisterInput) (*AuthResult, error) {
	panic("local customer adapter must not be selected")
}
func (poisonedCustomerAdapter) Login(context.Context, LoginInput) (*AuthResult, error) {
	panic("local customer adapter must not be selected")
}
func (poisonedCustomerAdapter) Validate(context.Context, string) (*CustomerContext, error) {
	panic("local customer adapter must not be selected")
}
func (poisonedCustomerAdapter) ResolveExternalIdentity(context.Context, ResolveExternalIdentityRequest) (*ExternalIdentityResolution, error) {
	panic("local customer adapter must not be selected")
}
func (poisonedCustomerAdapter) List(context.Context, string) ([]CustomerMembership, error) {
	panic("local customer adapter must not be selected")
}
func (poisonedCustomerAdapter) Resolve(context.Context, string, string) (*CustomerMembership, error) {
	panic("local customer adapter must not be selected")
}
