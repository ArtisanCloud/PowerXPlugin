package bootstrap

import (
	"context"
	"errors"
	"testing"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	sharedapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

type resolverDirectoryStub struct{}

func (resolverDirectoryStub) Mode() iamservice.IAMAdapterMode { return iamservice.IAMAdapterModeLocal }
func (resolverDirectoryStub) Login(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
	return nil, nil, errors.New("not used")
}
func (resolverDirectoryStub) Refresh(context.Context, string) (*iamservice.AuthTokens, error) {
	return nil, errors.New("not used")
}
func (resolverDirectoryStub) Logout(context.Context, string) error { return errors.New("not used") }
func (resolverDirectoryStub) CurrentUser(context.Context) (*iamservice.UserContext, error) {
	return nil, errors.New("not used")
}
func (resolverDirectoryStub) ListRoles(context.Context, string) ([]iamservice.RoleInfo, error) {
	return nil, nil
}
func (resolverDirectoryStub) ListDepartments(context.Context, string) ([]iamservice.DepartmentInfo, error) {
	return nil, nil
}
func (resolverDirectoryStub) ListMembers(context.Context, string) ([]iamservice.MemberInfo, error) {
	return nil, nil
}
func (resolverDirectoryStub) GetMember(context.Context, string, string) (*iamservice.MemberInfo, error) {
	return nil, iamservice.ErrMemberNotFound
}
func (resolverDirectoryStub) CheckPermission(context.Context, iamservice.TenantContext, string, string) error {
	return nil
}

func TestResolveAndBindFrameworkIAMLocal(t *testing.T) {
	resolver := NewProviderResolver(&config.Config{Context: &config.ContextConfig{ProviderMode: "local"}})
	deps := &sharedapp.Deps{IAMAdapterMode: iamservice.IAMAdapterModeLocal, IAMDirectory: resolverDirectoryStub{}}
	if err := ResolveAndBindFrameworkIAM(deps, resolver, fwiamadapters.NewRegistry()); err != nil {
		t.Fatalf("ResolveAndBindFrameworkIAM() error = %v", err)
	}
	if err := ValidateIAMBinding(deps); err != nil {
		t.Fatalf("ValidateIAMBinding() error = %v", err)
	}
}

func TestResolveAndBindFrameworkIAMRejectsModeMismatch(t *testing.T) {
	resolver := NewProviderResolver(&config.Config{Context: &config.ContextConfig{ProviderMode: "delegated"}})
	deps := &sharedapp.Deps{IAMAdapterMode: iamservice.IAMAdapterModeLocal, IAMDirectory: resolverDirectoryStub{}}
	if err := ResolveAndBindFrameworkIAM(deps, resolver, fwiamadapters.NewRegistry()); err == nil {
		t.Fatal("expected mode mismatch error")
	}
}

type resolverProxyStub struct{}

func (resolverProxyStub) Login(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) Refresh(context.Context, string) (*iamservice.AuthTokens, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) Logout(context.Context, string) error { return errors.New("not used") }
func (resolverProxyStub) MeContext(context.Context, string) (*authproxy.MeContext, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) GetDirectoryMember(context.Context, string) (*authproxy.DirectoryMember, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) BatchGetDirectoryMembers(context.Context, []string) ([]authproxy.DirectoryMember, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) BatchResolveDirectoryMembers(context.Context, []string) (*authproxy.DirectoryMemberResolution, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) BatchResolveDirectoryMembersByDisplayNames(context.Context, []string) (*authproxy.DirectoryMemberDisplayNameResolution, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) ListDirectoryDepartments(context.Context) ([]authproxy.DirectoryDepartment, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) ListDirectoryRoles(context.Context) ([]authproxy.DirectoryRole, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) ListDirectoryPermissions(context.Context) ([]authproxy.DirectoryPermission, error) {
	return nil, errors.New("not used")
}
func (resolverProxyStub) CheckDirectoryAuthorization(context.Context, authproxy.DirectoryAuthorizationRequest) (*authproxy.AuthorizationDecision, error) {
	return nil, errors.New("not used")
}

func TestResolveAndBindFrameworkIAMDelegated(t *testing.T) {
	resolver := NewProviderResolver(&config.Config{Context: &config.ContextConfig{ProviderMode: "delegated"}})
	deps := &sharedapp.Deps{IAMAdapterMode: iamservice.IAMAdapterModeDelegated, AuthProxy: resolverProxyStub{}}
	if err := ResolveAndBindFrameworkIAM(deps, resolver, fwiamadapters.NewRegistry()); err != nil {
		t.Fatalf("ResolveAndBindFrameworkIAM() error = %v", err)
	}
	if err := ValidateIAMBinding(deps); err != nil {
		t.Fatalf("ValidateIAMBinding() error = %v", err)
	}
}
