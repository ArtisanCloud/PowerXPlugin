package customer

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	contactfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type editContactStore struct {
	contactfw.Store
	updates int
	input   contactfw.UpdateContactInput
	created contactfw.CreateContactInput
}

func (s *editContactStore) Update(_ context.Context, in contactfw.UpdateContactInput) (*contactfw.Contact, error) {
	s.updates++
	s.input = in
	return &contactfw.Contact{ContactUUID: in.ContactUUID, CustomerUUID: in.CustomerUUID, Email: *in.Email, Phone: *in.Phone}, nil
}
func (s *editContactStore) Create(_ context.Context, in contactfw.CreateContactInput) (*contactfw.Contact, error) {
	s.created = in
	return &contactfw.Contact{ContactUUID: uuid.NewString(), CustomerUUID: in.CustomerUUID, Email: in.Email, Phone: in.Phone}, nil
}
func TestContactEditRoutesAndChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, route := range []string{"local", "delegated"} {
		t.Run(route, func(t *testing.T) {
			local, host := &editContactStore{}, &editContactStore{}
			localRuntime, _ := contactfw.NewRuntime(fwprovider.ModeLocal, local, nil)
			hostRuntime, _ := contactfw.NewRuntime(fwprovider.ModeDelegated, nil, host)
			h := NewContactHandler(&app.Deps{ContactDebugLocalRuntime: localRuntime, ContactDebugDelegatedRuntime: hostRuntime})
			customerUUID, contactUUID, tenantUUID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			selected, other := local, host
			if route == "delegated" {
				selected, other = host, local
			}
			for _, body := range []string{`{"email":"edit@example.test","phone":"+8613800000000"}`, `{"email":"","phone":""}`} {
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Params = gin.Params{{Key: "customerUUID", Value: customerUUID}, {Key: "contactUUID", Value: contactUUID}}
				c.Request = httptest.NewRequest(http.MethodPatch, "/?framework_debug_route="+route+"&tenant_uuid="+tenantUUID, bytes.NewBufferString(body))
				c.Request.Header.Set("Content-Type", "application/json")
				h.Update(c)
				if rec.Code != http.StatusOK || selected.input.Email == nil || selected.input.Phone == nil || selected.input.CustomerUUID != customerUUID || selected.input.ContactUUID != contactUUID || other.updates != 0 {
					t.Fatalf("status=%d input=%+v other=%d", rec.Code, selected.input, other.updates)
				}
			}
			if *selected.input.Email != "" || *selected.input.Phone != "" {
				t.Fatal("channel clear was lost")
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Params = gin.Params{{Key: "customerUUID", Value: customerUUID}}
			c.Request = httptest.NewRequest(http.MethodPost, "/?framework_debug_route="+route+"&tenant_uuid="+tenantUUID, bytes.NewBufferString(`{"display_name":"fixture","email":"create@example.test","phone":"12345","status":"active","creation_intent":"explicit_create"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.Create(c)
			if rec.Code != http.StatusCreated || selected.created.Email != "create@example.test" || selected.created.Phone != "12345" {
				t.Fatalf("status=%d input=%+v", rec.Code, selected.created)
			}
		})
	}
}

func TestCustomerDebugUpdateRejectsUnavailableHostAndInvalidRoute(t *testing.T) {
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeLocal})
	for route, want := range map[string]int{"delegated": http.StatusServiceUnavailable, "invalid": http.StatusBadRequest} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPatch, "/?framework_debug_route="+route, bytes.NewBufferString(`{}`))
		h.UpdateAccount(c)
		if rec.Code != want {
			t.Fatalf("route=%s status=%d", route, rec.Code)
		}
	}
}

func TestCustomerDebugUpdateUsesLocalEvenWhenStartupDelegated(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:customer-debug-edit?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(`CREATE TABLE customer_accounts (id INTEGER PRIMARY KEY, customer_uuid TEXT, tenant_uuid TEXT, display_name TEXT, primary_email TEXT, primary_phone TEXT, status TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	customerUUID, tenantUUID := uuid.NewString(), uuid.NewString()
	if err = db.Exec(`INSERT INTO customer_accounts (id,customer_uuid,tenant_uuid,display_name,status) VALUES (1,?,?,?,'active')`, customerUUID, tenantUUID, "before").Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeDelegated, DB: db})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "customerUUID", Value: customerUUID}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/?framework_debug_route=local&tenant_uuid="+tenantUUID, bytes.NewBufferString(`{"display_name":"after","primary_email":"after@example.test"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.UpdateAccount(c)
	var name string
	if err = db.Raw(`SELECT display_name FROM customer_accounts WHERE customer_uuid=?`, customerUUID).Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || name != "after" {
		t.Fatalf("status=%d name=%s body=%s", rec.Code, name, rec.Body.String())
	}
}

type editAccountInvoker struct {
	request gateway.InvokeRequest
	calls   int
}

func (s *editAccountInvoker) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.calls++
	s.request = req
	body := req.Payload.(map[string]any)["body"].(map[string]any)
	return &gateway.Response{Data: map[string]any{"payload": map[string]any{"item": map[string]any{"uuid": body["customer_uuid"], "type": "person", "primary_contact_uuid": uuid.NewString(), "display_name": "updated"}}}}, nil
}
func TestCustomerDebugUpdateUsesServiceAndRejectsUnsafeFields(t *testing.T) {
	stub := &editAccountInvoker{}
	client, _ := customerfw.NewAccountSelectorClient(stub)
	// No local database and no Admin client: only the service contract can succeed.
	h := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeLocal, CustomerAccountSelector: client})
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"primary_phone":""}`, http.StatusOK},
		{`{"type":"company"}`, http.StatusBadRequest},
		{`{"primary_phone":null}`, http.StatusBadRequest},
		{`{"tenant_uuid":"override"}`, http.StatusBadRequest},
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Params = gin.Params{{Key: "customerUUID", Value: uuid.NewString()}}
		c.Request = httptest.NewRequest(http.MethodPatch, "/?framework_debug_route=delegated", bytes.NewBufferString(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.UpdateAccount(c)
		if rec.Code != tc.want {
			t.Fatalf("body=%s status=%d response=%s", tc.body, rec.Code, rec.Body.String())
		}
	}
	if stub.calls != 1 || stub.request.CapabilityID != customerfw.CapabilityCustomerAccountsServiceManage {
		t.Fatalf("unexpected calls: %+v", stub)
	}
	body := stub.request.Payload.(map[string]any)["body"].(map[string]any)
	if len(body) != 3 || body["primary_phone"] != "" || body["operation"] != "update" {
		t.Fatalf("unexpected patch: %+v", body)
	}
}
