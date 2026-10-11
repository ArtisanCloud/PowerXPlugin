package customer

import (
	"context"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	audit "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/admin_console"
	m "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestIdentityManagementLocalContract(t *testing.T) {
	db := openCustomerIdentityTestDB(t, "management")
	require.NoError(t, db.Exec(`CREATE TABLE admin_console_audit_events (id TEXT PRIMARY KEY, plugin_id TEXT, tenant_uuid TEXT, actor_id TEXT, actor_name TEXT, actor_email TEXT, permission_code TEXT, action TEXT, resource_type TEXT, resource_ref TEXT, summary TEXT, diff TEXT, occurred_at DATETIME, created_at DATETIME)`).Error)
	store := repo.NewExternalIdentityStore(db, "com.powerx.test.shop")
	ctx := fw.WithTenantUUID(context.Background(), uuid.NewString())
	subject := fw.ShopifyExternalIdentitySubject("test.example", "gid://shopify/Customer/1")
	missing, err := store.Lookup(ctx, subject)
	require.NoError(t, err)
	require.False(t, missing.Found)
	for _, model := range []any{&m.CustomerAccount{}, &m.CustomerAuthIdentity{}, &m.CustomerTenantMembership{}, &m.Contact{}} {
		var n int64
		require.NoError(t, db.Model(model).Count(&n).Error)
		require.Zero(t, n)
	}
	// Separate adapter instances race against the same database and identity lock.
	var wg sync.WaitGroup
	results := make(chan *fw.ExternalIdentityItem, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, e := repo.NewExternalIdentityStore(db, "com.powerx.test.shop").CreateAndBind(ctx, subject, fw.ExternalIdentityCustomer{PrimaryEmail: "only@example.test"})
			results <- out
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var first *fw.ExternalIdentityItem
	for item := range results {
		if first == nil {
			first = item
		}
		require.Equal(t, first.IdentityUUID, item.IdentityUUID)
		require.Equal(t, first.CustomerUUID, item.CustomerUUID)
	}
	for _, model := range []any{&m.CustomerAccount{}, &m.CustomerAuthIdentity{}, &m.CustomerTenantMembership{}, &m.Contact{}, &audit.AuditEvent{}} {
		var n int64
		require.NoError(t, db.Model(model).Count(&n).Error)
		require.EqualValues(t, 1, n)
	}
	bound, err := store.Bind(ctx, first.CustomerUUID, subject)
	require.NoError(t, err)
	require.Equal(t, first, bound)
	list, err := store.ListByCustomer(ctx, first.CustomerUUID, 1, 20)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	second, err := store.CreateAndBind(ctx, "shop:test.example:customer:2", fw.ExternalIdentityCustomer{DisplayName: "second"})
	require.NoError(t, err)
	_, err = store.Bind(ctx, second.CustomerUUID, subject)
	require.Equal(t, 409, fw.HTTPStatus(err))
	found, err := store.Lookup(ctx, subject)
	require.NoError(t, err)
	require.Equal(t, first.CustomerUUID, found.Item.CustomerUUID)
	other := fw.WithTenantUUID(context.Background(), uuid.NewString())
	invisible, err := store.Lookup(other, subject)
	require.NoError(t, err)
	require.False(t, invisible.Found)
	_, err = store.Bind(other, first.CustomerUUID, "shop:test.example:customer:3")
	require.Equal(t, 404, fw.HTTPStatus(err))
	_, err = store.CreateAndBind(other, subject, fw.ExternalIdentityCustomer{DisplayName: "must not create"})
	require.Equal(t, 409, fw.HTTPStatus(err))
	var account m.CustomerAccount
	require.NoError(t, db.Where("customer_uuid = ?", first.CustomerUUID).First(&account).Error)
	require.Empty(t, account.GivenName)
	require.Empty(t, account.Nickname)
	// Failing the last audit write must roll back every newly created business object.
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_identity_audit BEFORE INSERT ON admin_console_audit_events BEGIN SELECT RAISE(ABORT, 'fixture_failure'); END`).Error)
	_, err = store.CreateAndBind(ctx, "shop:test.example:customer:rollback", fw.ExternalIdentityCustomer{DisplayName: "rollback"})
	require.Error(t, err)
	for _, model := range []any{&m.CustomerAccount{}, &m.CustomerAuthIdentity{}, &m.CustomerTenantMembership{}, &m.Contact{}} {
		var n int64
		require.NoError(t, db.Model(model).Count(&n).Error)
		require.EqualValues(t, 2, n)
	}
}
