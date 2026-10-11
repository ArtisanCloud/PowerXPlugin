package customer

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	audit "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/admin_console"
	m "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type primaryFixture struct {
	db               *gorm.DB
	ctx              context.Context
	tenant, customer string
	store            *FrameworkContactLocalStore
	manager          *FrameworkPrimaryContactManager
	a, b             string
}

func primaryTestFixture(t *testing.T) *primaryFixture {
	t.Helper()
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	for _, ddl := range []string{
		`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT UNIQUE, status TEXT, tenant_uuid TEXT);`,
		`CREATE TABLE customer_tenant_memberships (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, membership_uuid TEXT UNIQUE, tenant_uuid TEXT, customer_uuid TEXT, primary_contact_uuid TEXT, status TEXT, roles TEXT, scopes TEXT, source TEXT, expires_at DATETIME, metadata TEXT, UNIQUE(tenant_uuid,customer_uuid));`,
		`CREATE TABLE customer_contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, contact_uuid TEXT UNIQUE, tenant_uuid TEXT, customer_uuid TEXT, display_name TEXT, given_name TEXT, family_name TEXT, email TEXT, phone TEXT, status TEXT, roles TEXT, tags TEXT, metadata TEXT);`,
		`CREATE TABLE admin_console_audit_events (id TEXT PRIMARY KEY, plugin_id TEXT, tenant_uuid TEXT, actor_id TEXT, actor_name TEXT, actor_email TEXT, permission_code TEXT, action TEXT, resource_type TEXT, resource_ref TEXT, summary TEXT, diff TEXT, occurred_at DATETIME, created_at DATETIME);`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	f := &primaryFixture{db: db, tenant: uuid.NewString(), customer: uuid.NewString(), store: NewFrameworkContactLocalStore(db), manager: NewFrameworkPrimaryContactManager(repository.NewContactRepository(db), "fixture.plugin")}
	f.ctx = customerfw.WithTenantUUID(context.Background(), f.tenant)
	if err := db.Table("customer_accounts").Create(map[string]any{"customer_uuid": f.customer, "tenant_uuid": f.tenant, "status": m.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&m.CustomerTenantMembership{MembershipUUID: uuid.NewString(), TenantUUID: f.tenant, CustomerUUID: f.customer, Status: m.StatusActive, PrimaryContactUUID: ""}).Error; err != nil {
		t.Fatal(err)
	}
	for _, out := range []*string{&f.a, &f.b} {
		row, err := f.store.Create(f.ctx, fw.CreateContactInput{CustomerUUID: f.customer, DisplayName: "fixture", Status: fw.StatusActive, CreationIntent: fw.CreationIntentExplicitCreate, Roles: []fw.Role{fw.RoleLegalRepresentative}})
		if err != nil {
			t.Fatal(err)
		}
		*out = row.ContactUUID
	}
	return f
}
func (f *primaryFixture) set(target, expected string) (*fw.PrimaryContactResult, error) {
	return f.manager.SetPrimary(f.ctx, fw.SetPrimaryContactInput{CustomerUUID: f.customer, ContactUUID: target, ExpectedPrimaryContactUUID: &expected})
}
func (f *primaryFixture) primary(t *testing.T) string {
	t.Helper()
	var row m.CustomerTenantMembership
	if err := f.db.Where("tenant_uuid = ? AND customer_uuid = ?", f.tenant, f.customer).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row.PrimaryContactUUID
}
func (f *primaryFixture) checkRoles(t *testing.T, expected string) {
	t.Helper()
	var rows []m.Contact
	if err := f.db.Where("tenant_uuid = ? AND customer_uuid = ?", f.tenant, f.customer).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range rows {
		var roles []fw.Role
		if err := json.Unmarshal(row.Roles, &roles); err != nil {
			t.Fatal(err)
		}
		for _, role := range roles {
			if role == fw.RolePrimary {
				count++
				if row.ContactUUID != expected || row.Status != string(fw.StatusActive) {
					t.Fatal("CONTACT_PRIMARY_INVARIANT_INVALID")
				}
			}
		}
	}
	if count != 1 || f.primary(t) != expected {
		t.Fatal("CONTACT_PRIMARY_REFERENCE_INVALID")
	}
}

func TestPrimaryLocalTransactionsAndProtectedUpdates(t *testing.T) {
	f := primaryTestFixture(t)
	if _, err := f.set(f.a, ""); err != nil {
		t.Fatal(err)
	}
	f.checkRoles(t, f.a)
	if _, err := f.set(f.b, ""); fw.CodeOf(err) != fw.CodePrimaryChanged {
		t.Fatal("CONTACT_PRIMARY_STALE_ACCEPTED", err)
	}
	inactive := fw.StatusInactive
	if _, err := f.store.Update(f.ctx, fw.UpdateContactInput{CustomerUUID: f.customer, ContactUUID: f.a, Status: &inactive}); fw.CodeOf(err) != fw.CodePrimaryChangeRequired {
		t.Fatal("CONTACT_PRIMARY_DEACTIVATION_ACCEPTED", err)
	}
	empty := []fw.Role{}
	if _, err := f.store.Update(f.ctx, fw.UpdateContactInput{CustomerUUID: f.customer, ContactUUID: f.a, Roles: &empty}); fw.CodeOf(err) != fw.CodePrimaryOperationRequired {
		t.Fatal("CONTACT_PRIMARY_ROLE_REMOVAL_ACCEPTED", err)
	}
	primary := []fw.Role{fw.RolePrimary}
	if _, err := f.store.Update(f.ctx, fw.UpdateContactInput{CustomerUUID: f.customer, ContactUUID: f.b, Roles: &primary}); fw.CodeOf(err) != fw.CodePrimaryOperationRequired {
		t.Fatal("CONTACT_PRIMARY_ROLE_ONLY_ACCEPTED", err)
	}
	if _, err := f.set(f.b, f.a); err != nil {
		t.Fatal(err)
	}
	f.checkRoles(t, f.b)
	ref, err := repository.NewContactRepository(f.db).PrimaryReference(f.ctx, f.tenant, f.customer)
	if err != nil || ref != f.b {
		t.Fatal("CONTACT_PRIMARY_READBACK_INVALID", err)
	}
	if _, err := f.set(f.b, f.b); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := f.db.Model(&audit.AuditEvent{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatal("CONTACT_PRIMARY_AUDIT_COUNT_INVALID", err)
	}
	if _, err := f.store.Update(f.ctx, fw.UpdateContactInput{CustomerUUID: f.customer, ContactUUID: f.a, Status: &inactive}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.set(f.a, f.b); fw.CodeOf(err) != fw.CodePrimaryNotActive {
		t.Fatal("CONTACT_PRIMARY_INACTIVE_ACCEPTED", err)
	}
	f.checkRoles(t, f.b)
}

func TestPrimaryLocalOwnershipAndAuditRollback(t *testing.T) {
	f := primaryTestFixture(t)
	if _, err := f.set(f.a, ""); err != nil {
		t.Fatal(err)
	}
	for _, tenant := range []string{f.tenant, uuid.NewString()} {
		id := uuid.NewString()
		row := m.Contact{ContactUUID: id, TenantUUID: tenant, CustomerUUID: uuid.NewString(), DisplayName: "fixture", Status: "active", Roles: []byte(`[]`), Tags: []byte(`[]`), Metadata: []byte(`{}`)}
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		_, err := f.set(id, f.a)
		if fw.CodeOf(err) != fw.CodeNotFound && fw.CodeOf(err) != fw.CodeCustomerMismatch {
			t.Fatal("CONTACT_PRIMARY_FOREIGN_ACCEPTED", err)
		}
	}
	if err := f.db.Callback().Create().Before("gorm:create").Register("fixture_primary_audit_failure", func(db *gorm.DB) {
		if db.Statement.Table == models.TableAdminConsoleAuditEvents {
			db.AddError(errors.New("CONTACT_PRIMARY_AUDIT_REJECTED"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err := f.set(f.b, f.a)
	if fw.CodeOf(err) != fw.CodePrimaryOperationFailed {
		t.Fatal("CONTACT_PRIMARY_AUDIT_FAILURE_LOST", err)
	}
	f.checkRoles(t, f.a)
	var count int64
	f.db.Model(&audit.AuditEvent{}).Count(&count)
	if count != 1 {
		t.Fatal("CONTACT_PRIMARY_AUDIT_ROLLBACK_INVALID")
	}
}

func TestPrimarySetAndDeactivateSerialize(t *testing.T) {
	f := primaryTestFixture(t)
	if _, err := f.set(f.a, ""); err != nil {
		t.Fatal(err)
	}
	var setErr, updateErr error
	var wg sync.WaitGroup
	wg.Add(2)
	start := make(chan struct{})
	go func() { defer wg.Done(); <-start; _, setErr = f.set(f.b, f.a) }()
	go func() {
		defer wg.Done()
		<-start
		inactive := fw.StatusInactive
		_, updateErr = f.store.Update(f.ctx, fw.UpdateContactInput{CustomerUUID: f.customer, ContactUUID: f.b, Status: &inactive})
	}()
	close(start)
	wg.Wait()
	if setErr == nil {
		if fw.CodeOf(updateErr) != fw.CodePrimaryChangeRequired {
			t.Fatal("CONTACT_PRIMARY_DEACTIVATION_RACE", updateErr)
		}
		f.checkRoles(t, f.b)
	} else {
		if fw.CodeOf(setErr) != fw.CodePrimaryNotActive || updateErr != nil {
			t.Fatal("CONTACT_PRIMARY_SET_RACE", setErr, updateErr)
		}
		f.checkRoles(t, f.a)
	}
}
