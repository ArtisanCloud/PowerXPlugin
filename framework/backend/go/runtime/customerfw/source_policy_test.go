package customerfw

import "testing"

func TestSourcePolicyBlocksLocalAndMockInProduction(t *testing.T) {
	for _, source := range []CustomerAuthSource{CustomerAuthSourceLocal, CustomerAuthSourceLocalDev, CustomerAuthSourceMock} {
		err := ValidateSourcePolicy(SourcePolicy{Mode: source, Production: true})
		if CodeOf(err) != CodeCustomerIdentitySourceBlocked {
			t.Fatalf("expected source blocked for %s, got %v", source, err)
		}
	}
}

func TestSourcePolicyAllowsProductionBreakGlass(t *testing.T) {
	err := ValidateSourcePolicy(SourcePolicy{Mode: CustomerAuthSourceLocalDev, Production: true, BreakGlass: true})
	if err != nil {
		t.Fatalf("expected break-glass allow, got %v", err)
	}
}
