package customer

import (
	"context"
	"testing"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	"gorm.io/gorm"
)

func TestLocalExternalIdentityResolverCreatesAndReusesTenantMembership(t *testing.T) {
	db := openCustomerIdentityTestDB(t, "create-reuse")
	resolver, err := NewLocalExternalIdentityResolver(db, "com.powerx.test.shop")
	if err != nil {
		t.Fatalf("NewLocalExternalIdentityResolver(): %v", err)
	}
	ctx := customerfw.WithTenantUUID(context.Background(), "tenant-a")
	first, err := resolver.ResolveExternalIdentity(ctx, customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"})
	if err != nil {
		t.Fatalf("ResolveExternalIdentity(create): %v", err)
	}
	second, err := resolver.ResolveExternalIdentity(ctx, customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Changed name must not create another identity"})
	if err != nil {
		t.Fatalf("ResolveExternalIdentity(reuse): %v", err)
	}
	if first.CustomerUUID == "" || first.MembershipUUID == "" || first.Type != "person" || first.PrimaryContactUUID == "" {
		t.Fatalf("resolution UUIDs are required: %#v", first)
	}
	if first.CustomerUUID != second.CustomerUUID || first.MembershipUUID != second.MembershipUUID || first.PrimaryContactUUID != second.PrimaryContactUUID {
		t.Fatalf("identity resolution must be idempotent: first=%#v second=%#v", first, second)
	}
	var memberships int64
	if err := db.Model(&customermodel.CustomerTenantMembership{}).Where("tenant_uuid = ? AND customer_uuid = ?", "tenant-a", first.CustomerUUID).Count(&memberships).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if memberships != 1 {
		t.Fatalf("membership count = %d, want 1", memberships)
	}
	var contacts int64
	if err := db.Model(&customermodel.Contact{}).Where("tenant_uuid = ? AND customer_uuid = ?", "tenant-a", first.CustomerUUID).Count(&contacts).Error; err != nil || contacts != 1 {
		t.Fatalf("contacts=%d err=%v", contacts, err)
	}
}

func TestLocalExternalIdentityResolverRequiresTrustedContextTenant(t *testing.T) {
	db := openCustomerIdentityTestDB(t, "tenant-required")
	resolver, err := NewLocalExternalIdentityResolver(db, "com.powerx.test.shop")
	if err != nil {
		t.Fatalf("NewLocalExternalIdentityResolver(): %v", err)
	}
	_, err = resolver.ResolveExternalIdentity(context.Background(), customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"})
	if customerfw.CodeOf(err) != customerfw.CodeCustomerTenantRequired {
		t.Fatalf("error code = %s, want %s; err=%v", customerfw.CodeOf(err), customerfw.CodeCustomerTenantRequired, err)
	}
}

func TestLocalExternalIdentityResolverRollsBackWhenContactCannotBeCreated(t *testing.T) {
	db := openCustomerIdentityTestDB(t, "contact-failure")
	if err := db.Exec("DROP TABLE customer_contacts").Error; err != nil {
		t.Fatal(err)
	}
	resolver, err := NewLocalExternalIdentityResolver(db, "com.powerx.test.shop")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.ResolveExternalIdentity(customerfw.WithTenantUUID(context.Background(), "tenant-a"), customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"})
	if err == nil {
		t.Fatal("contact failure must fail the identity resolution")
	}
	for _, table := range []string{"customer_accounts", "customer_auth_identities", "customer_tenant_memberships"} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%s rows=%d err=%v", table, count, err)
		}
	}
}

func TestLocalExternalIdentityResolverRepairsMissingPrimaryContactIdempotently(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		t.Run(map[bool]string{true: "reuse", false: "create"}[reuse], func(t *testing.T) {
			db := openCustomerIdentityTestDB(t, "repair-"+map[bool]string{true: "reuse", false: "create"}[reuse])
			resolver, err := NewLocalExternalIdentityResolver(db, "com.powerx.test.shop")
			if err != nil {
				t.Fatal(err)
			}
			ctx := customerfw.WithTenantUUID(context.Background(), "tenant-a")
			input := customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"}
			first, err := resolver.ResolveExternalIdentity(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&customermodel.CustomerTenantMembership{}).Where("membership_uuid = ?", first.MembershipUUID).Update("primary_contact_uuid", "").Error; err != nil {
				t.Fatal(err)
			}
			if !reuse {
				if err := db.Where("contact_uuid = ?", first.PrimaryContactUUID).Delete(&customermodel.Contact{}).Error; err != nil {
					t.Fatal(err)
				}
			}
			second, err := resolver.ResolveExternalIdentity(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			third, err := resolver.ResolveExternalIdentity(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if second.CustomerUUID != first.CustomerUUID || second.PrimaryContactUUID != third.PrimaryContactUUID || second.PrimaryContactUUID == "" {
				t.Fatalf("first=%+v second=%+v third=%+v", first, second, third)
			}
			if reuse && second.PrimaryContactUUID != first.PrimaryContactUUID {
				t.Fatalf("existing contact changed: first=%+v second=%+v", first, second)
			}
			var count int64
			if err := db.Model(&customermodel.Contact{}).Where("tenant_uuid = ? AND customer_uuid = ?", "tenant-a", first.CustomerUUID).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("contacts=%d err=%v", count, err)
			}
		})
	}
}

func TestLocalExternalIdentityResolverRejectsInactiveMembership(t *testing.T) {
	db := openCustomerIdentityTestDB(t, "membership-inactive")
	resolver, err := NewLocalExternalIdentityResolver(db, "com.powerx.test.shop")
	if err != nil {
		t.Fatalf("NewLocalExternalIdentityResolver(): %v", err)
	}
	ctx := customerfw.WithTenantUUID(context.Background(), "tenant-a")
	resolved, err := resolver.ResolveExternalIdentity(ctx, customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"})
	if err != nil {
		t.Fatalf("ResolveExternalIdentity(create): %v", err)
	}
	if err := db.Model(&customermodel.CustomerTenantMembership{}).Where("membership_uuid = ?", resolved.MembershipUUID).Update("status", customermodel.StatusDisabled).Error; err != nil {
		t.Fatalf("disable membership: %v", err)
	}
	_, err = resolver.ResolveExternalIdentity(ctx, customerfw.ResolveExternalIdentityRequest{ProviderSubject: "shop:test.example:customer:1", DisplayName: "Customer One"})
	if customerfw.CodeOf(err) != customerfw.CodeCustomerMembershipDisabled {
		t.Fatalf("error code = %s, want %s; err=%v", customerfw.CodeOf(err), customerfw.CodeCustomerMembershipDisabled, err)
	}
}

func openCustomerIdentityTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	ddls := []string{
		`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT NOT NULL UNIQUE, type TEXT, primary_contact_uuid TEXT, primary_email TEXT, primary_phone TEXT, display_name TEXT, nickname TEXT, given_name TEXT, family_name TEXT, avatar_url TEXT, locale TEXT, timezone TEXT, status TEXT NOT NULL, email TEXT, phone TEXT, password_hash TEXT, email_verified INTEGER, phone_verified INTEGER, metadata TEXT, tenant_uuid TEXT);`,
		`CREATE TABLE customer_auth_identities (identity_uuid TEXT UNIQUE, id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT NOT NULL, provider TEXT NOT NULL, provider_subject TEXT NOT NULL, email TEXT, phone TEXT, password_hash TEXT, status TEXT NOT NULL, verified_at DATETIME, metadata TEXT, UNIQUE(provider, provider_subject));`,
		`CREATE TABLE customer_tenant_memberships (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, membership_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, primary_contact_uuid TEXT, status TEXT NOT NULL, roles TEXT, scopes TEXT, source TEXT NOT NULL, expires_at DATETIME, metadata TEXT, UNIQUE(tenant_uuid, customer_uuid));`,
		`CREATE TABLE customer_contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, contact_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, display_name TEXT NOT NULL, given_name TEXT, family_name TEXT, email TEXT, phone TEXT, status TEXT NOT NULL, roles TEXT, tags TEXT, metadata TEXT);`,
	}
	for _, ddl := range ddls {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create customer test table: %v", err)
		}
	}
	return db
}
