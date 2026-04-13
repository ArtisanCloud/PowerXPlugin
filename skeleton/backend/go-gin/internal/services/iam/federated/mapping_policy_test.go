package federated

import "testing"

func TestMappingServiceRecomputeOnlyOnVersionChange(t *testing.T) {
	svc := NewMappingService()
	svc.Upsert(MappingPolicy{TenantUUID: "tenant-a", Version: "v1", Roles: []string{"r1"}})

	first := svc.ApplyOnLogin("tenant-a", 100)
	if !first.Recomputed || first.Version != "v1" {
		t.Fatalf("first apply = %+v", first)
	}

	second := svc.ApplyOnLogin("tenant-a", 100)
	if second.Recomputed {
		t.Fatalf("second apply recomputed=true, want false")
	}

	svc.Upsert(MappingPolicy{TenantUUID: "tenant-a", Version: "v2", Roles: []string{"r2"}})
	third := svc.ApplyOnLogin("tenant-a", 100)
	if !third.Recomputed || third.Version != "v2" {
		t.Fatalf("third apply = %+v", third)
	}
}
