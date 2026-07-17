package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/gin-gonic/gin"
)

type metadataInvokerStub struct {
	called bool
	last   gateway.InvokeRequest
}

func (s *metadataInvokerStub) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.called = true
	s.last = req
	return &gateway.Response{
		TraceID: "trace-metadata",
		Status:  "ok",
		Data: map[string]any{
			"payload": map[string]any{
				"items": []any{
					map[string]any{
						"uuid":         "dict-1",
						"namespace":    "customer_status",
						"module":       "customer",
						"name_i18n":    map[string]any{"zh-CN": "客户状态"},
						"status":       "active",
						"display_name": "客户状态",
					},
				},
				"pagination": map[string]any{"total": 1, "page": 1, "page_size": 20},
			},
		},
	}, nil
}

func TestMetadataHandlerDelegatedUsesFrameworkClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &metadataInvokerStub{}
	client, err := fwmetadata.NewClient(fwmetadata.Config{Invoker: invoker})
	if err != nil {
		t.Fatalf("metadata client: %v", err)
	}
	h := &Handler{mode: fwprovider.ModeDelegated, delegated: client}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metadata/dictionaries?page=1&page_size=20", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.ListDictionaryNamespaces(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !invoker.called {
		t.Fatal("expected delegated metadata handler to call framework gateway invoker")
	}
	if invoker.last.CapabilityID != fwmetadata.CapabilityDictionaryRead {
		t.Fatalf("capability=%s", invoker.last.CapabilityID)
	}
}

func TestMetadataHandlerLocalDoesNotCallDelegatedClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &metadataInvokerStub{}
	client, err := fwmetadata.NewClient(fwmetadata.Config{Invoker: invoker})
	if err != nil {
		t.Fatalf("metadata client: %v", err)
	}
	h := &Handler{mode: fwprovider.ModeLocal, delegated: client}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metadata/dictionaries?tenant_uuid=tenant-a", nil)
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	h.ListDictionaryNamespaces(c)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if invoker.called {
		t.Fatal("local metadata handler must not call delegated framework gateway invoker")
	}
}
