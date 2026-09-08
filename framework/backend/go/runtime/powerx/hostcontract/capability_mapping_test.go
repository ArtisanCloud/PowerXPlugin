package hostcontract

import "testing"

func TestPublishedCapabilityMappings(t *testing.T) {
	for _, tc := range []struct {
		module    Module
		operation Operation
		id        string
	}{
		{ModuleIAM, OperationIAMTenant, "com.corex.iam.directory.read"},
		{ModuleIAM, OperationIAMMembersList, "com.corex.iam.directory.read"},
		{ModuleIAM, OperationIAMMembersBatchResolve, "com.corex.iam.directory.read"},
		{ModuleIAM, OperationIAMMembersResolveDisplayNames, "com.corex.iam.directory.read"},
		{ModuleIAM, OperationIAMMemberGet, "com.corex.iam.members.read"},
		{ModuleIAM, OperationIAMMembersBatchGet, "com.corex.iam.members.read"},
		{ModuleAgent, OperationAgentHealthSummary, "com.corex.agent.lifecycle.manage"},
		{ModuleAgent, OperationAgentFreeze, "com.corex.agent.lifecycle.manage"},
		{ModuleAgent, OperationAgentRecover, "com.corex.agent.lifecycle.manage"},
		{ModuleAgent, OperationAgentRebalance, "com.corex.agent.lifecycle.manage"},
		{ModuleNotifications, OperationNotificationCreate, "com.corex.notifications.create"},
	} {
		d, err := LookupOperation(tc.module, tc.operation)
		if err != nil || d.CapabilityID != tc.id {
			t.Fatalf("%s/%s descriptor=%+v err=%v", tc.module, tc.operation, d, err)
		}
	}
}
