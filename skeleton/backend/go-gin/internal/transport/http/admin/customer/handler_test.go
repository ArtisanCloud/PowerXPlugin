package customer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type customerInvokerStub struct {
	called bool
	last   gateway.InvokeRequest
}

func (s *customerInvokerStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.called = true
	s.last = req
	return &gateway.Response{
		TraceID: "trace-customer",
		Status:  "ok",
		Data: map[string]any{
			"payload": map[string]any{
				"items": []any{
					map[string]any{"customer_uuid": "customer-a", "display_name": "Customer A", "status": "active"},
				},
				"page": 1, "page_size": 20, "total": 1,
			},
		},
	}, nil
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
