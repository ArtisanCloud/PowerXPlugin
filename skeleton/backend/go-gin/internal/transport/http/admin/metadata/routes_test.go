package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/gin-gonic/gin"
)

func TestMetadataHandlerDelegatedUsesFrameworkClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/v1/tenant/metadata/dictionaries" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer service-sts" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":{"payload":{"items":[{"uuid":"11111111-1111-4111-8111-111111111111","namespace":"customer_status","module":"customer","name_i18n":{"zh-CN":"客户状态"},"status":"active"}],"pagination":{"total":1,"page":1,"page_size":20}}}}`))
	}))
	defer server.Close()
	client, err := fwmetadata.NewHostClientWithTokenProvider(fwmetadata.HostClientConfig{BaseURL: server.URL}, fwmetadata.HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
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
	if !called {
		t.Fatal("expected delegated metadata handler to call tenant Host Contract")
	}
}

func TestMetadataHandlerLocalDoesNotCallDelegatedClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client, err := fwmetadata.NewHostClientWithTokenProvider(fwmetadata.HostClientConfig{BaseURL: server.URL}, fwmetadata.HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
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
	if called {
		t.Fatal("local metadata handler must not call delegated Metadata Host Contract")
	}
}
