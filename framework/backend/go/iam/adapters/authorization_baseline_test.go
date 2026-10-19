package adapters

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

type baselineStub struct {
	result *contracts.AuthorizationBaselineResult
}

func (s *baselineStub) ReadMemberBaselines(context.Context, contracts.AuthorizationBaselineQuery) (*contracts.AuthorizationBaselineResult, error) {
	return s.result, nil
}

func TestRegistryOptionalAuthorizationBaselines(t *testing.T) {
	for _, provider := range []contracts.AuthorizationBaselineService{nil, (*baselineStub)(nil), &baselineStub{}} {
		registry := NewRegistry()
		if _, err := registry.AuthorizationBaselines(); !iamerrors.IsCode(err, iamerrors.CodeAdapterNotBound) {
			t.Fatal(err)
		}
		if err := registry.Bind(contracts.IAMAdapterModeLocal, Bundle{Directory: stubDirectory{}, Authz: stubAuthz{}, Context: stubIdentityContext{}, AuthorizationBaselines: provider}); err != nil {
			t.Fatal(err)
		}
		got, err := registry.AuthorizationBaselines()
		if nilAdapter(provider) {
			if got != nil || !iamerrors.IsCode(err, iamerrors.CodeAdapterNotBound) {
				t.Fatal(got, err)
			}
		} else if got != provider || err != nil {
			t.Fatal(got, err)
		}
	}
}

func TestInjectedLocalBaselineResultIsValidated(t *testing.T) {
	provider, err := ValidateAuthorizationBaselines("com.powerx.fixture", &baselineStub{})
	if err != nil {
		t.Fatal(err)
	}
	query := contracts.AuthorizationBaselineQuery{TenantUUID: "20000000-0000-4000-8000-000000000001", MemberUUIDs: []string{"10000000-0000-4000-8000-000000000001"}, PermissionCodes: []string{"orders:read"}}
	if result, err := provider.ReadMemberBaselines(context.Background(), query); result != nil || !iamerrors.IsCode(err, iamerrors.CodeBaselineResponseInvalid) {
		t.Fatal(result, err)
	}
	if _, err := ValidateAuthorizationBaselines("com.powerx.fixture", (*baselineStub)(nil)); !iamerrors.IsCode(err, iamerrors.CodeAdapterNotBound) {
		t.Fatal(err)
	}
}
