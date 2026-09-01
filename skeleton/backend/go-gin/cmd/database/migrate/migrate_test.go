package migrate

import (
	"context"
	"fmt"
	"testing"

	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	identitymodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestMigratePluginModelsIncludesFederatedIAMTables(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() {
		EntityModels.ForceSchemaForTests("public")
	})

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}

	mustHave := []interface{}{
		&identitymodel.Role{},
		&identitymodel.Department{},
		&identitymodel.Permission{},
		&identitymodel.FederatedExternalIdentity{},
		&identitymodel.FederatedBinding{},
		&identitymodel.FederatedLoginChallenge{},
		&identitymodel.FederatedRiskEvent{},
	}
	for _, model := range mustHave {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("HasTable(%T) = false, want true", model)
		}
	}
	for _, table := range []string{identitymodel.Role{}.TableName(), identitymodel.Department{}.TableName(), identitymodel.Permission{}.TableName()} {
		if !db.Migrator().HasColumn(table, "uuid") {
			t.Fatalf("%s.uuid = missing", table)
		}
	}
	for table, column := range map[string]string{
		identitymodel.Member{}.TableName():         "department_uuid",
		identitymodel.Department{}.TableName():     "parent_department_uuid",
		identitymodel.MemberRole{}.TableName():     "member_uuid",
		identitymodel.RolePermission{}.TableName(): "permission_uuid",
	} {
		if !db.Migrator().HasColumn(table, column) {
			t.Fatalf("%s.%s = missing", table, column)
		}
	}
}

func TestBackfillIAMUUIDRelations(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam_uuid_backfill?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}
	tenant := "tenant-uuid-backfill"
	parent := identitymodel.Department{}
	parent.TenantUuid, parent.Name, parent.Code, parent.Path = tenant, "Parent", "parent", "parent"
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := identitymodel.Department{}
	child.TenantUuid, child.Name, child.Code, child.Path = tenant, "Child", "child", "parent.child"
	child.ParentID = &parent.ID
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	account := identitymodel.User{Email: "backfill@example.test", DisplayName: "Backfill", PasswordHash: "hash"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	member := identitymodel.Member{UserID: account.ID, Username: "backfill", Status: identitymodel.StatusActive, DepartmentID: &child.ID}
	member.TenantUuid = tenant
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	role := identitymodel.Role{Code: "backfill-role", Name: "Backfill role", ScopeType: identitymodel.RoleScopeTenant}
	role.TenantUuid = tenant
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	permission := identitymodel.Permission{Description: "Backfill permission", Resource: "backfill", Action: "read"}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatal(err)
	}
	memberRole := identitymodel.MemberRole{UserID: member.ID, RoleID: role.ID}
	if err := db.Create(&memberRole).Error; err != nil {
		t.Fatal(err)
	}
	rolePermission := identitymodel.RolePermission{RoleID: role.ID, PermissionID: permission.ID}
	if err := db.Create(&rolePermission).Error; err != nil {
		t.Fatal(err)
	}
	// Simulate rows created before UUID columns existed.
	for index, entry := range []struct {
		table string
		id    uint64
	}{
		{identitymodel.Department{}.TableName(), parent.ID}, {identitymodel.Department{}.TableName(), child.ID},
		{identitymodel.Member{}.TableName(), member.ID}, {identitymodel.Role{}.TableName(), role.ID},
		{identitymodel.Permission{}.TableName(), permission.ID},
	} {
		if err := db.Table(entry.table).Where("id = ?", entry.id).Update("uuid", fmt.Sprintf("legacy-%d", index)).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := backfillIAMUUIDRelations(context.Background(), db); err != nil {
		t.Fatalf("backfillIAMUUIDRelations() error = %v", err)
	}
	var gotChild identitymodel.Department
	if err := db.First(&gotChild, child.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotParent identitymodel.Department
	if err := db.First(&gotParent, parent.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotChild.UUID == "" || gotParent.UUID == "" || gotChild.ParentDepartmentUUID == nil || *gotChild.ParentDepartmentUUID != gotParent.UUID {
		t.Fatalf("parent uuid relation was not backfilled: child=%#v parent=%#v", gotChild, gotParent)
	}
	var gotMember identitymodel.Member
	if err := db.First(&gotMember, member.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotMember.UUID == "" || gotMember.DepartmentUUID == nil || *gotMember.DepartmentUUID != gotChild.UUID {
		t.Fatalf("department uuid relation was not backfilled: %#v", gotMember)
	}
	var gotMemberRole identitymodel.MemberRole
	if err := db.First(&gotMemberRole, "member_id = ? AND role_id = ?", member.ID, role.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotRole identitymodel.Role
	if err := db.First(&gotRole, role.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotRole.UUID == "" || gotMemberRole.MemberUUID != gotMember.UUID || gotMemberRole.RoleUUID != gotRole.UUID {
		t.Fatalf("member role uuid relation not backfilled: %#v", gotMemberRole)
	}
	var gotRolePermission identitymodel.RolePermission
	if err := db.First(&gotRolePermission, "role_id = ? AND permission_id = ?", role.ID, permission.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotPermission identitymodel.Permission
	if err := db.First(&gotPermission, permission.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotPermission.UUID == "" || gotRolePermission.RoleUUID != gotRole.UUID || gotRolePermission.PermissionUUID != gotPermission.UUID {
		t.Fatalf("role permission uuid relation not backfilled: %#v", gotRolePermission)
	}
}

func TestEnsureIAMIdentityUUIDsBackfillsBeforeAutoMigrate(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam_pre_migrate_uuid?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tables := []string{
		identitymodel.User{}.TableName(), identitymodel.Member{}.TableName(), identitymodel.Role{}.TableName(),
		identitymodel.Department{}.TableName(), identitymodel.Permission{}.TableName(),
	}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, uuid TEXT NULL)", table)).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(fmt.Sprintf("INSERT INTO %s (id, uuid) VALUES (1, NULL)", table)).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := ensureIAMIdentityUUIDs(context.Background(), db); err != nil {
		t.Fatalf("ensureIAMIdentityUUIDs() error = %v", err)
	}
	for _, table := range tables {
		var count int64
		if err := db.Table(table).Where("uuid IS NULL OR trim(uuid) = ''").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d empty uuid rows", table, count)
		}
	}
}
