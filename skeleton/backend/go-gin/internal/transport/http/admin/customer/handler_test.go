package customer

import (
	"bytes"
	"context"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type customerInvokerStub struct {
	called bool
	last   gateway.InvokeRequest
}

func (s *customerInvokerStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.called = true
	s.last = req
	if req.CapabilityID == customerfw.CapabilityCustomerAccountsServiceManage {
		return &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": "11111111-1111-4111-8111-111111111111", "display_name": "Customer A", "status": "active"}}}}, nil
	}
	return &gateway.Response{
		TraceID: "trace-customer",
		Status:  "ok",
		Data: map[string]any{
			"payload": map[string]any{
				"items": []any{
					map[string]any{"uuid": "11111111-1111-4111-8111-111111111111", "display_name": "Customer A", "status": "active"},
				},
				"page": 1, "page_size": 20, "total": 1,
			},
		},
	}, nil
}

func TestCustomerHandlerDebugCreateUsesTypedManageCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &customerInvokerStub{}
	client, err := customerfw.NewAccountSelectorClient(invoker)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeLocal, CustomerAccountSelector: client})
	rec := httptest.NewRecorder()
	body := []byte(`{"display_name":"Customer A","status":"active"}`)
	req := httptest.NewRequest(http.MethodPost, "/customers/debug/basic-accounts?framework_debug_route=delegated&tenant_uuid="+uuid.NewString(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	h.CreateBasicAccount(c)
	if rec.Code != http.StatusCreated || invoker.last.CapabilityID != customerfw.CapabilityCustomerAccountsServiceManage || invoker.last.PreferredProtocol != "core_internal" {
		t.Fatalf("status=%d request=%+v body=%s", rec.Code, invoker.last, rec.Body.String())
	}
}

func TestCustomerHandlerDebugCreateRejectsCallerTenantInBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &customerInvokerStub{}
	client, err := customerfw.NewAccountSelectorClient(invoker)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{CustomerAccountSelector: client})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customers/debug/basic-accounts?framework_debug_route=delegated&tenant_uuid="+uuid.NewString(), bytes.NewBufferString(`{"display_name":"Customer A","tenant_uuid":"`+uuid.NewString()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	h.CreateBasicAccount(c)
	if rec.Code != http.StatusBadRequest || invoker.called {
		t.Fatalf("status=%d called=%v body=%s", rec.Code, invoker.called, rec.Body.String())
	}
}

func TestCustomerHandlerDebugLocalCreateAddsMembershipWithoutIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:customer-debug-create?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT UNIQUE NOT NULL, tenant_uuid TEXT NOT NULL, status TEXT NOT NULL, primary_email TEXT, primary_phone TEXT, display_name TEXT, nickname TEXT, given_name TEXT, family_name TEXT, avatar_url TEXT, locale TEXT, timezone TEXT, metadata TEXT, email TEXT, phone TEXT, password_hash TEXT, email_verified BOOLEAN, phone_verified BOOLEAN);`,
		`CREATE TABLE customer_tenant_memberships (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, membership_uuid TEXT UNIQUE NOT NULL, tenant_uuid TEXT NOT NULL, customer_uuid TEXT NOT NULL, status TEXT NOT NULL, roles TEXT, scopes TEXT, source TEXT, expires_at DATETIME, metadata TEXT);`,
		`CREATE TABLE customer_auth_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, customer_uuid TEXT, provider TEXT, provider_subject TEXT, email TEXT, phone TEXT, password_hash TEXT, status TEXT, verified_at DATETIME, metadata TEXT);`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeDelegated, DB: db})
	tenantUUID := uuid.NewString()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customers/debug/basic-accounts?framework_debug_route=local&tenant_uuid="+tenantUUID, bytes.NewBufferString(`{"display_name":"Local Customer","status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	h.CreateBasicAccount(c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var accounts, memberships, identities int64
	if err := db.Model(&customermodel.CustomerAccount{}).Where("tenant_uuid = ?", tenantUUID).Count(&accounts).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&customermodel.CustomerTenantMembership{}).Where("tenant_uuid = ?", tenantUUID).Count(&memberships).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&customermodel.CustomerAuthIdentity{}).Count(&identities).Error; err != nil {
		t.Fatal(err)
	}
	if accounts != 1 || memberships != 1 || identities != 0 {
		t.Fatalf("accounts=%d memberships=%d identities=%d", accounts, memberships, identities)
	}
}

func TestCustomerHandlerDelegatedListUsesFrameworkClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &customerInvokerStub{}
	client, err := customerfw.NewAdminClient(customerfw.AdminClientConfig{Invoker: invoker})
	if err != nil {
		t.Fatalf("customer admin client: %v", err)
	}
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeDelegated, CustomerAdmin: client})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/accounts?tenant_uuid=tenant-a", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.ListAccounts(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !invoker.called {
		t.Fatal("expected delegated customer handler to call framework gateway invoker")
	}
	if invoker.last.CapabilityID != customerfw.CapabilityCustomerAccountsAdminManage {
		t.Fatalf("capability=%s", invoker.last.CapabilityID)
	}
}

func TestCustomerHandlerDebugDelegatedListDoesNotReadLocalProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &customerInvokerStub{}
	client, err := customerfw.NewAccountSelectorClient(invoker)
	if err != nil {
		t.Fatalf("customer account selector: %v", err)
	}
	// The plugin itself starts local; only the Framework Lab probe selects the
	// delegated route. A nil DB proves this handler cannot fall back locally.
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeLocal, CustomerAccountSelector: client})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/accounts?framework_debug_route=delegated", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.ListAccounts(c)

	if rec.Code != http.StatusOK || !invoker.called {
		t.Fatalf("status=%d called=%v body=%s", rec.Code, invoker.called, rec.Body.String())
	}
	if invoker.last.CapabilityID != customerfw.CapabilityCustomerAccountsServiceRead || invoker.last.PreferredProtocol != "core_internal" {
		t.Fatalf("unexpected selector invocation: %+v", invoker.last)
	}
}

func TestCustomerHandlerLocalMissingStoreReturnsProviderUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeLocal})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/accounts?tenant_uuid=tenant-a", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.ListAccounts(c)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
