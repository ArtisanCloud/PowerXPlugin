package federated

import "testing"

func TestJITPolicyServiceSwitch(t *testing.T) {
	svc := NewJITPolicyService()
	defaultPolicy := svc.Get("tenant-a")
	if !defaultPolicy.Enabled || defaultPolicy.Mode != JITPolicyUniqueMatch {
		t.Fatalf("default policy = %+v", defaultPolicy)
	}

	svc.Set(JITPolicy{TenantUUID: "tenant-a", Enabled: false, Mode: JITPolicyAdminOnly})
	if svc.AllowAutoBind("tenant-a") {
		t.Fatalf("AllowAutoBind() = true, want false")
	}

	svc.Set(JITPolicy{TenantUUID: "tenant-a", Enabled: true, Mode: JITPolicyUniqueMatch})
	if !svc.AllowAutoBind("tenant-a") {
		t.Fatalf("AllowAutoBind() = false, want true")
	}
}
