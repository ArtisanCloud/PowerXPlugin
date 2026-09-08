package iam

import (
	"context"
	"testing"
)

func TestTenantServiceUpdateByUUIDUsesOnlyTenantUUIDAtBusinessBoundary(t *testing.T) {
	db := newRoleServiceTestDB(t)
	tenant := seedTestTenant(t, db, "tenant-update-by-uuid")
	service := NewTenantService(db, NewAuditService(db))

	updated, err := service.UpdateByUUID(context.Background(), tenant.UUID, UpdateTenantInput{Name: "Updated tenant"})
	if err != nil {
		t.Fatalf("UpdateByUUID() error: %v", err)
	}
	if updated.UUID != tenant.UUID || updated.Name != "Updated tenant" {
		t.Fatalf("UpdateByUUID() = %#v", updated)
	}
}
