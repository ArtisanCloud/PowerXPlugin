package knowledge

import "testing"

func TestValidateSourcePolicyBlocksLocalProduction(t *testing.T) {
	err := ValidateSourcePolicy(SourcePolicy{Mode: ProviderModeLocal, Production: true})
	if CodeOf(err) != CodeForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if err := ValidateSourcePolicy(SourcePolicy{Mode: ProviderModeLocal, Production: true, BreakGlass: true}); err != nil {
		t.Fatalf("break-glass should allow local: %v", err)
	}
	if err := ValidateSourcePolicy(SourcePolicy{Mode: ProviderModeDelegated, Production: true}); err != nil {
		t.Fatalf("delegated should pass: %v", err)
	}
}
