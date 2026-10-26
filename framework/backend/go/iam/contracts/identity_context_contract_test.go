package contracts_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	iamcontext "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
)

func TestIdentityContextCarriesExplicitRootStatus(t *testing.T) {
	for _, root := range []bool{false, true} {
		data, err := json.Marshal(contracts.IdentityContext{IsRoot: root})
		if err != nil || !strings.Contains(string(data), `"is_root":`) {
			t.Fatalf("root status must be explicit: %s %v", data, err)
		}
		var value contracts.IdentityContext
		if err := json.Unmarshal(data, &value); err != nil || value.IsRoot != root {
			t.Fatalf("root identity round trip: %v", err)
		}
	}
}

func TestClaimOnlyIdentityResolverDoesNotPromoteRoot(t *testing.T) {
	// This legacy decoder does not authenticate the token; it must never turn
	// unverified claims or role names into the privileged adapter assertion.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"tid":"11111111-1111-4111-8111-111111111111","is_root":true,"roles":["root","super_admin"]}`))
	value, err := (iamcontext.IdentityContextResolver{}).ResolveIdentity(context.Background(), "fixture."+payload+".fixture")
	if err != nil || value.IsRoot {
		t.Fatalf("claim-only resolver promoted root: %v", err)
	}
}
