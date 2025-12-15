package iam

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"github.com/google/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoleService_CreateCloneAndManagePermissions(t *testing.T) {
	db := newRoleServiceTestDB(t)
	ctx := context.Background()
	audit := NewAuditService(db)
	svc := NewRoleService(db, audit, "test.plugin")

	tenant := seedTestTenant(t, db, "tenant-test")
	perms := seedTestPermissions(t, db, []string{"iam.member:read", "iam.member:write", "iam.role:read"})

	// prepare base role with two permissions
	baseRole := &iamm.Role{
		BaseModel:     basemodels.BaseModel{TenantUuid: tenant.UUID},
		Code:          "base.role",
		Name:          "Base Role",
		ScopeType:     iamm.RoleScopeTenant,
		PolicyVersion: "pv-base",
	}
	if err := db.Create(baseRole).Error; err != nil {
		t.Fatalf("create base role: %v", err)
	}
	if err := replaceRolePermissionsTx(ctx, db, baseRole, []uint64{perms[0].ID, perms[1].ID}, baseRole.PolicyVersion); err != nil {
		t.Fatalf("seed base role permissions: %v", err)
	}

	view, err := svc.Create(ctx, CreateRoleInput{
		TenantUUID:  tenant.UUID,
		Code:        "custom.role",
		Name:        "Custom",
		ScopeType:   iamm.RoleScopeTenant,
		CloneRoleID: &baseRole.ID,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if len(view.PermissionIDs) != 2 {
		t.Fatalf("expected cloned permissions, got %v", view.PermissionIDs)
	}

	originalVersion := view.PolicyVersion

	updated, err := svc.ReplacePermissions(ctx, ReplaceRolePermissionsInput{
		RoleID:        view.ID,
		TenantUUID:    tenant.UUID,
		PermissionIDs: []uint64{perms[2].ID},
	})
	if err != nil {
		t.Fatalf("replace permissions: %v", err)
	}
	if updated.PolicyVersion == originalVersion {
		t.Fatalf("policy version should change after replace")
	}
	if len(updated.PermissionIDs) != 1 || updated.PermissionIDs[0] != perms[2].ID {
		t.Fatalf("unexpected permission ids: %v", updated.PermissionIDs)
	}

	var rp iamm.RolePermission
	if err := db.Where("role_id = ?", view.ID).First(&rp).Error; err != nil {
		t.Fatalf("load role_permission: %v", err)
	}
	if rp.TenantUuid != tenant.UUID {
		t.Fatalf("role_permission tenant mismatch, want %s got %s", tenant.UUID, rp.TenantUuid)
	}
}

func TestRoleService_AssignMembers(t *testing.T) {
	db := newRoleServiceTestDB(t)
	ctx := context.Background()
	svc := NewRoleService(db, NewAuditService(db), "test.plugin")
	tenant := seedTestTenant(t, db, "tenant-assign")
	members := seedTestMembers(t, db, tenant, 2)

	role := &iamm.Role{
		BaseModel:     basemodels.BaseModel{TenantUuid: tenant.UUID},
		Code:          "assign.role",
		Name:          "Assign Role",
		ScopeType:     iamm.RoleScopeTenant,
		PolicyVersion: "pv-initial",
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	err := svc.AddMembers(ctx, RoleMembersInput{
		RoleID:     role.ID,
		TenantUUID: tenant.UUID,
		MemberIDs:  []uint64{members[0].ID, members[1].ID},
	})
	if err != nil {
		t.Fatalf("add members: %v", err)
	}
	var count int64
	if err := db.Model(&iamm.MemberRole{}).Where("role_id = ?", role.ID).Count(&count).Error; err != nil {
		t.Fatalf("count user role: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 role members, got %d", count)
	}

	err = svc.RemoveMembers(ctx, RoleMembersInput{
		RoleID:     role.ID,
		TenantUUID: tenant.UUID,
		MemberIDs:  []uint64{members[0].ID},
	})
	if err != nil {
		t.Fatalf("remove members: %v", err)
	}
	if err := db.Model(&iamm.MemberRole{}).Where("role_id = ?", role.ID).Count(&count).Error; err != nil {
		t.Fatalf("count user role after remove: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 role member after removal, got %d", count)
	}
}

const testSQLiteDriver = "iam_role_service_sqlite"

var registerSQLiteOnce sync.Once

func registerSQLiteDriver() {
	registerSQLiteOnce.Do(func() {
		sql.Register(testSQLiteDriver, &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("gen_random_uuid", func() string {
					return strings.ToLower(uuid.NewString())
				}, true)
			},
		})
	})
}

func newRoleServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	registerSQLiteDriver()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: testSQLiteDriver,
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIAMTablesForTest(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db
}

func createIAMTablesForTest(db *gorm.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS iam_tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT,
			key TEXT,
			name TEXT,
			status TEXT,
			plan TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT,
			phone TEXT,
			display_name TEXT,
			avatar_url TEXT,
			status TEXT,
			password_hash TEXT,
			meta TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_uuid TEXT,
			user_id INTEGER,
			username TEXT,
			display_name TEXT,
			avatar_url TEXT,
			status TEXT,
			department_id INTEGER,
			meta TEXT,
			last_login_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_uuid TEXT,
			code TEXT,
			name TEXT,
			description TEXT,
			scope_type TEXT,
			policy_version TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource TEXT,
			action TEXT,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_member_roles (
			member_id INTEGER,
			role_id INTEGER,
			created_at DATETIME,
			PRIMARY KEY (member_id, role_id)
		)`,
		`CREATE TABLE IF NOT EXISTS iam_role_permissions (
			role_id INTEGER,
			permission_id INTEGER,
			tenant_uuid TEXT,
			policy_version TEXT,
			created_at DATETIME,
			PRIMARY KEY (role_id, permission_id)
		)`,
		`CREATE TABLE IF NOT EXISTS iam_refresh_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT,
			user_id INTEGER,
			tenant_uuid TEXT,
			member_id INTEGER,
			expires_at DATETIME,
			revoked BOOLEAN,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS iam_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_uuid TEXT,
			actor_member_id INTEGER,
			action TEXT,
			resource TEXT,
			diff TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedTestTenant(t *testing.T, db *gorm.DB, key string) *iamm.Tenant {
	t.Helper()
	tenant := &iamm.Tenant{
		UUID:   key + "-uuid",
		Key:    key,
		Name:   "Tenant " + key,
		Status: iamm.StatusActive,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant
}

func seedTestPermissions(t *testing.T, db *gorm.DB, values []string) []*iamm.Permission {
	t.Helper()
	result := make([]*iamm.Permission, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, ":")
		res := parts[0]
		act := "read"
		if len(parts) > 1 {
			act = parts[1]
		}
		perm := &iamm.Permission{Resource: res, Action: act, Description: value}
		if err := db.Create(perm).Error; err != nil {
			t.Fatalf("create permission %s: %v", value, err)
		}
		result = append(result, perm)
	}
	return result
}

func seedTestMembers(t *testing.T, db *gorm.DB, tenant *iamm.Tenant, total int) []*iamm.Member {
	t.Helper()
	members := make([]*iamm.Member, 0, total)
	for i := 0; i < total; i++ {
		account := &iamm.User{
			Email:        fmt.Sprintf("member%d@example.com", i),
			PasswordHash: "hash",
			Status:       iamm.StatusActive,
		}
		if err := db.Create(account).Error; err != nil {
			t.Fatalf("create account: %v", err)
		}
		member := &iamm.Member{
			BaseModel:    basemodels.BaseModel{TenantUuid: tenant.UUID},
			UserID:       account.ID,
			Username:     fmt.Sprintf("member-%d", i),
			Status:       iamm.StatusActive,
			DepartmentID: nil,
		}
		if err := db.Create(member).Error; err != nil {
			t.Fatalf("create member: %v", err)
		}
		members = append(members, member)
	}
	return members
}
