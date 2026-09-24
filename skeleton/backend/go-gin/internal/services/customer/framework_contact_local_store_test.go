package customer

import (
	"context"
	"testing"

	contactfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	metadatamodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/metadata"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestFrameworkContactLocalStoreScopesAndNeverCreatesOnResolveMiss(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:framework-contact-store?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT NOT NULL UNIQUE, status TEXT NOT NULL, tenant_uuid TEXT);`,
		`CREATE TABLE customer_tenant_memberships (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, membership_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, status TEXT NOT NULL, roles TEXT, scopes TEXT, source TEXT, expires_at DATETIME, metadata TEXT, UNIQUE(tenant_uuid, customer_uuid));`,
		`CREATE TABLE customer_contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, contact_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, display_name TEXT NOT NULL, given_name TEXT, family_name TEXT, status TEXT NOT NULL, roles TEXT NOT NULL, tags TEXT NOT NULL, metadata TEXT NOT NULL, UNIQUE(tenant_uuid, customer_uuid, contact_uuid));`,
		`CREATE TABLE customer_contact_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, identity_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, contact_uuid TEXT NOT NULL, channel_dictionary_item_uuid TEXT, external_subject TEXT NOT NULL, status TEXT NOT NULL, verified_at DATETIME, metadata TEXT NOT NULL, UNIQUE(tenant_uuid, channel_dictionary_item_uuid, external_subject));`,
		`CREATE TABLE metadata_dictionary_namespaces (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, namespace TEXT NOT NULL, module TEXT NOT NULL, name_i18n TEXT, description_i18n TEXT, status TEXT NOT NULL, UNIQUE(tenant_uuid, namespace));`,
		`CREATE TABLE metadata_dictionary_items (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, namespace_uuid TEXT NOT NULL, code TEXT NOT NULL, label_i18n TEXT, description_i18n TEXT, status TEXT NOT NULL, sort_order INTEGER NOT NULL DEFAULT 0, reference_count INTEGER NOT NULL DEFAULT 0, UNIQUE(namespace_uuid, code));`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	tenantUUID, customerUUID, otherCustomerUUID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, customerUUID := range []string{customerUUID, otherCustomerUUID} {
		if err := db.Table("customer_accounts").Create(map[string]any{"customer_uuid": customerUUID, "tenant_uuid": tenantUUID, "status": customermodel.StatusActive}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Table("customer_tenant_memberships").Create(map[string]any{"membership_uuid": uuid.NewString(), "tenant_uuid": tenantUUID, "customer_uuid": customerUUID, "status": customermodel.StatusActive}).Error; err != nil {
			t.Fatal(err)
		}
	}
	namespaceUUID, channelDictionaryItemUUID := uuid.NewString(), uuid.NewString()
	if err := db.Create(&metadatamodel.DictionaryNamespace{UUID: namespaceUUID, TenantUUID: tenantUUID, Namespace: contactIdentityChannelNamespace, Module: "corex.customer", Status: "enabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&metadatamodel.DictionaryItem{UUID: channelDictionaryItemUUID, TenantUUID: tenantUUID, NamespaceUUID: namespaceUUID, Code: "email", Status: "enabled"}).Error; err != nil {
		t.Fatal(err)
	}
	store := NewFrameworkContactLocalStore(db)
	ctx := customerfw.WithTenantUUID(context.Background(), tenantUUID)
	created, err := store.Create(ctx, contactfw.CreateContactInput{CustomerUUID: customerUUID, DisplayName: "Ada", Status: contactfw.StatusActive, Roles: []contactfw.Role{contactfw.RolePrimary}, Tags: []string{"vip"}, CreationIntent: contactfw.CreationIntentExplicitCreate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveIdentity(ctx, contactfw.ResolveContactIdentityInput{CustomerUUID: customerUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID, ExternalSubject: "ada@example.test"}); contactfw.CodeOf(err) != contactfw.CodeIdentityNotFound {
		t.Fatalf("miss code=%s err=%v", contactfw.CodeOf(err), err)
	}
	var identityCount int64
	if err := db.Model(&customermodel.ContactIdentity{}).Count(&identityCount).Error; err != nil || identityCount != 0 {
		t.Fatalf("identity count=%d err=%v", identityCount, err)
	}
	if _, err := store.BindIdentity(ctx, contactfw.BindContactIdentityInput{CustomerUUID: customerUUID, ContactUUID: created.ContactUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID, ExternalSubject: "ada@example.test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, contactfw.GetContactInput{CustomerUUID: otherCustomerUUID, ContactUUID: created.ContactUUID}); contactfw.CodeOf(err) != contactfw.CodeCustomerMismatch {
		t.Fatalf("cross customer code=%s err=%v", contactfw.CodeOf(err), err)
	}
	resolved, err := store.ResolveIdentity(ctx, contactfw.ResolveContactIdentityInput{CustomerUUID: customerUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID, ExternalSubject: "ada@example.test"})
	if err != nil || resolved.Contact.ContactUUID != created.ContactUUID {
		t.Fatalf("resolved=%+v err=%v", resolved, err)
	}
	legacy := &customermodel.ContactIdentity{IdentityUUID: uuid.NewString(), TenantUUID: tenantUUID, CustomerUUID: customerUUID, ContactUUID: created.ContactUUID, ExternalSubject: "legacy@example.test", Status: string(contactfw.IdentityStatusActive), Metadata: []byte("{}")}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveIdentity(ctx, contactfw.ResolveContactIdentityInput{CustomerUUID: customerUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID, ExternalSubject: legacy.ExternalSubject}); contactfw.CodeOf(err) != contactfw.CodeIdentityChannelMigrationRequired {
		t.Fatalf("legacy resolve code=%s err=%v", contactfw.CodeOf(err), err)
	}
	migrated, err := store.MigrateIdentityChannel(ctx, contactfw.MigrateContactIdentityChannelInput{CustomerUUID: customerUUID, ContactUUID: created.ContactUUID, IdentityUUID: legacy.IdentityUUID, ChannelDictionaryItemUUID: channelDictionaryItemUUID})
	if err != nil || migrated.ChannelDictionaryItemUUID != channelDictionaryItemUUID {
		t.Fatalf("migrated=%+v err=%v", migrated, err)
	}
}

func TestFrameworkContactLocalStoreRejectsUnknownChannelDictionaryItem(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:framework-contact-channel-invalid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT NOT NULL UNIQUE, status TEXT NOT NULL, tenant_uuid TEXT);`,
		`CREATE TABLE customer_tenant_memberships (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, membership_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, status TEXT NOT NULL, roles TEXT, scopes TEXT, source TEXT, expires_at DATETIME, metadata TEXT, UNIQUE(tenant_uuid, customer_uuid));`,
		`CREATE TABLE customer_contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, contact_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, display_name TEXT NOT NULL, given_name TEXT, family_name TEXT, status TEXT NOT NULL, roles TEXT NOT NULL, tags TEXT NOT NULL, metadata TEXT NOT NULL, UNIQUE(tenant_uuid, customer_uuid, contact_uuid));`,
		`CREATE TABLE customer_contact_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, identity_uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, contact_uuid TEXT NOT NULL, channel_dictionary_item_uuid TEXT, external_subject TEXT NOT NULL, status TEXT NOT NULL, verified_at DATETIME, metadata TEXT NOT NULL, UNIQUE(tenant_uuid, channel_dictionary_item_uuid, external_subject));`,
		`CREATE TABLE metadata_dictionary_namespaces (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, namespace TEXT NOT NULL, module TEXT NOT NULL, name_i18n TEXT, description_i18n TEXT, status TEXT NOT NULL, UNIQUE(tenant_uuid, namespace));`,
		`CREATE TABLE metadata_dictionary_items (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, uuid TEXT NOT NULL UNIQUE, tenant_uuid TEXT NOT NULL, namespace_uuid TEXT NOT NULL, code TEXT NOT NULL, label_i18n TEXT, description_i18n TEXT, status TEXT NOT NULL, sort_order INTEGER NOT NULL DEFAULT 0, reference_count INTEGER NOT NULL DEFAULT 0, UNIQUE(namespace_uuid, code));`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	tenantUUID, customerUUID := uuid.NewString(), uuid.NewString()
	if err := db.Table("customer_accounts").Create(map[string]any{"customer_uuid": customerUUID, "tenant_uuid": tenantUUID, "status": customermodel.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("customer_tenant_memberships").Create(map[string]any{"membership_uuid": uuid.NewString(), "tenant_uuid": tenantUUID, "customer_uuid": customerUUID, "status": customermodel.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	store := NewFrameworkContactLocalStore(db)
	ctx := customerfw.WithTenantUUID(context.Background(), tenantUUID)
	contact, err := store.Create(ctx, contactfw.CreateContactInput{CustomerUUID: customerUUID, DisplayName: "Ada", Status: contactfw.StatusActive, CreationIntent: contactfw.CreationIntentExplicitCreate})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.BindIdentity(ctx, contactfw.BindContactIdentityInput{CustomerUUID: customerUUID, ContactUUID: contact.ContactUUID, ChannelDictionaryItemUUID: uuid.NewString(), ExternalSubject: "ada@example.test"})
	if contactfw.CodeOf(err) != contactfw.CodeChannelDictionaryInvalid {
		t.Fatalf("code=%s err=%v", contactfw.CodeOf(err), err)
	}
}
