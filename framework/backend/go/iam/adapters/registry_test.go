package adapters

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

type stubDirectory struct{}

func (stubDirectory) GetTenant(context.Context, string) (*contracts.Tenant, error) {
	return &contracts.Tenant{TenantUUID: "tenant-1"}, nil
}
func (stubDirectory) ListDepartments(context.Context, string) ([]contracts.Department, error) {
	return nil, nil
}
func (stubDirectory) ListMembers(context.Context, string) ([]contracts.Member, error) {
	return nil, nil
}
func (stubDirectory) ListRoles(context.Context, string) ([]contracts.Role, error) {
	return nil, nil
}
func (stubDirectory) ListPermissions(context.Context, string) ([]contracts.Permission, error) {
	return nil, nil
}

type stubAuthz struct{}

func (stubAuthz) Authorize(context.Context, contracts.AuthorizationRequest) (*contracts.AuthorizationDecision, error) {
	return &contracts.AuthorizationDecision{Allowed: true}, nil
}

type stubIdentityContext struct{}

func (stubIdentityContext) ResolveIdentity(context.Context, string) (*contracts.IdentityContext, error) {
	return &contracts.IdentityContext{TenantUUID: "tenant-1"}, nil
}

func TestRegistry_BindAndResolve(t *testing.T) {
	registry := NewRegistry()
	if registry.IsBound() {
		t.Fatalf("expected registry to be unbound initially")
	}

	err := registry.Bind(contracts.IAMAdapterModeLocal, Bundle{
		Directory: stubDirectory{},
		Authz:     stubAuthz{},
		Context:   stubIdentityContext{},
	})
	if err != nil {
		t.Fatalf("unexpected bind error: %v", err)
	}
	if !registry.IsBound() {
		t.Fatalf("expected registry to be bound")
	}
	mode, ok := registry.Mode()
	if !ok || mode != contracts.IAMAdapterModeLocal {
		t.Fatalf("expected local mode, got mode=%q ok=%v", mode, ok)
	}
	if _, err := registry.Directory(); err != nil {
		t.Fatalf("expected directory service, got err=%v", err)
	}
	if _, err := registry.Authz(); err != nil {
		t.Fatalf("expected authz service, got err=%v", err)
	}
	if _, err := registry.IdentityContext(); err != nil {
		t.Fatalf("expected identity context service, got err=%v", err)
	}
}

func TestRegistry_BindFailWhenAlreadyBound(t *testing.T) {
	registry := NewRegistry()
	first := Bundle{Directory: stubDirectory{}, Authz: stubAuthz{}, Context: stubIdentityContext{}}
	if err := registry.Bind(contracts.IAMAdapterModeLocal, first); err != nil {
		t.Fatalf("unexpected first bind error: %v", err)
	}

	err := registry.Bind(contracts.IAMAdapterModeDelegated, first)
	if err == nil {
		t.Fatalf("expected second bind error")
	}
	if !iamerrors.IsCode(err, iamerrors.CodeAdapterAlreadyBind) {
		t.Fatalf("expected code %s, got %s", iamerrors.CodeAdapterAlreadyBind, iamerrors.CodeOf(err))
	}
	if got := iamerrors.StatusCode(err); got != 409 {
		t.Fatalf("expected status 409, got %d", got)
	}
}

func TestRegistry_ReadFailWhenUnbound(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Directory()
	if err == nil {
		t.Fatalf("expected error when reading unbound directory")
	}
	if !iamerrors.IsCode(err, iamerrors.CodeAdapterNotBound) {
		t.Fatalf("expected code %s, got %s", iamerrors.CodeAdapterNotBound, iamerrors.CodeOf(err))
	}
	if got := iamerrors.StatusCode(err); got != 424 {
		t.Fatalf("expected status 424, got %d", got)
	}
}
