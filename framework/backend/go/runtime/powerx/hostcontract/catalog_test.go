package hostcontract

import "testing"

func TestLookupOperationReturnsCapabilityAndRisk(t *testing.T) {
	operation, err := LookupOperation(ModuleIAM, OperationIAMMembersList)
	if err != nil {
		t.Fatalf("lookup IAM members list: %v", err)
	}
	if operation.CapabilityID != "com.corex.iam.directory.read" {
		t.Fatalf("capability_id = %q", operation.CapabilityID)
	}
	if !operation.ReadOnly || operation.RequiresConfirmation {
		t.Fatalf("IAM members list risk flags = read_only:%t confirmation:%t", operation.ReadOnly, operation.RequiresConfirmation)
	}

	operation, err = LookupOperation(ModuleKnowledge, OperationKnowledgeDocumentCreate)
	if err != nil {
		t.Fatalf("lookup Knowledge document create: %v", err)
	}
	if operation.ReadOnly || !operation.RequiresConfirmation {
		t.Fatalf("Knowledge document create risk flags = read_only:%t confirmation:%t", operation.ReadOnly, operation.RequiresConfirmation)
	}
}

func TestLookupOperationRejectsUnsupportedModuleAndOperation(t *testing.T) {
	if _, err := LookupOperation(Module("unknown"), "status"); CodeOf(err) != ReasonUnsupportedModule {
		t.Fatalf("unsupported module reason = %q", CodeOf(err))
	}
	if _, err := LookupOperation(ModuleIAM, "unknown"); CodeOf(err) != ReasonUnsupportedOperation {
		t.Fatalf("unsupported operation reason = %q", CodeOf(err))
	}
}

func TestValidateNoTenantOverrideRecursesThroughProbeInput(t *testing.T) {
	if err := ValidateNoTenantOverride(map[string]any{"member_uuid": "member-1"}); err != nil {
		t.Fatalf("safe input rejected: %v", err)
	}
	if err := ValidateNoTenantOverride(map[string]any{"filters": map[string]any{"tenant_uuid": "tenant-a"}}); CodeOf(err) != ReasonTenantOverrideForbidden {
		t.Fatalf("nested tenant override reason = %q", CodeOf(err))
	}
}

func TestProbeResultPreservesAuditFieldsAndReasonCode(t *testing.T) {
	result := ProbeResult{
		Module:       ModuleIAM,
		Operation:    OperationIAMMembersList,
		ProviderMode: "delegated",
		CapabilityID: "com.corex.iam.members.read",
		TraceID:      "trace-1",
		ReasonCode:   ReasonForbidden,
	}
	if result.Audit().CapabilityID != result.CapabilityID || result.Audit().ReasonCode != ReasonForbidden {
		t.Fatalf("audit fields not preserved: %#v", result.Audit())
	}
}
