package iam

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/gorm"
)

func TestLocalDirectoryBatchResolveMembersByDisplayNamesIsTenantScopedAndAmbiguitySafe(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:iam_display_name_resolution?mode=memory&cache=shared&_time_format=sqlite"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&iamm.Tenant{}, &iamm.User{}, &iamm.Member{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	tenantA := iamm.Tenant{UUID: "tenant-a", Key: "tenant-a", Name: "Tenant A", Status: iamm.StatusActive}
	tenantB := iamm.Tenant{UUID: "tenant-b", Key: "tenant-b", Name: "Tenant B", Status: iamm.StatusActive}
	if err := db.Create(&tenantA).Error; err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	if err := db.Create(&tenantB).Error; err != nil {
		t.Fatalf("create tenant B: %v", err)
	}
	createMember := func(tenantUUID, memberUUID, userUUID, displayName string) {
		t.Helper()
		user := iamm.User{UUID: userUUID, Email: userUUID + "@example.test", DisplayName: displayName, Status: iamm.StatusActive, PasswordHash: "not-used"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %s: %v", userUUID, err)
		}
		member := iamm.Member{BaseModel: models.BaseModel{TenantUuid: tenantUUID}, UUID: memberUUID, UserID: user.ID, Username: userUUID, DisplayName: displayName, Status: iamm.StatusActive}
		if err := db.Create(&member).Error; err != nil {
			t.Fatalf("create member %s: %v", memberUUID, err)
		}
	}
	createMember("tenant-a", "member-alpha", "user-alpha", "Alpha")
	createMember("tenant-a", "member-beta-1", "user-beta-1", "Beta")
	createMember("tenant-a", "member-beta-2", "user-beta-2", "Beta")
	createMember("tenant-b", "member-gamma", "user-gamma", "Gamma")

	directory, err := NewLocalDirectory(db, &config.Config{Context: &config.ContextConfig{}})
	if err != nil {
		t.Fatalf("NewLocalDirectory() error = %v", err)
	}
	result, err := directory.BatchResolveMembersByDisplayNames(context.Background(), "tenant-a", []string{" Alpha ", "Unknown", "Beta", "Alpha", "Gamma"})
	if err != nil {
		t.Fatalf("BatchResolveMembersByDisplayNames() error = %v", err)
	}
	if len(result) != 5 || result[0].Status != MemberDisplayNameFound || result[0].Member == nil || result[0].Member.MemberUUID != "member-alpha" || result[1].Status != MemberDisplayNameNotFound || result[2].Status != MemberDisplayNameAmbiguous || result[2].Member != nil || result[3].Status != MemberDisplayNameFound || result[4].Status != MemberDisplayNameNotFound {
		t.Fatalf("result = %#v", result)
	}
}
