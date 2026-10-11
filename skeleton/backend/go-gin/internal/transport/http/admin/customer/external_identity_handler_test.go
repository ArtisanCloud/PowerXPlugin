package customer

import (
	"bytes"
	"context"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

type identityProbeStore struct {
	fw.ExternalIdentityStore
	calls   int
	failure error
}

func (s *identityProbeStore) Lookup(ctx context.Context, subject string) (*fw.ExternalIdentityLookup, error) {
	s.calls++
	return &fw.ExternalIdentityLookup{Found: false}, s.failure
}
func TestIdentityDebugRouteIsolationAndErrors(t *testing.T) {
	local, host := &identityProbeStore{}, &identityProbeStore{}
	lr, _ := fw.NewRuntime(provider.ModeLocal, fw.RuntimeAdapters{Identities: local}, fw.RuntimeAdapters{})
	hr, _ := fw.NewRuntime(provider.ModeDelegated, fw.RuntimeAdapters{}, fw.RuntimeAdapters{Identities: host})
	h := NewHandler(&app.Deps{CustomerIdentityDebugLocal: lr, CustomerIdentityDebugDelegated: hr})
	for _, route := range []string{"local", "delegated"} {
		for _, status := range []int{200, 403, 409, 424} {
			selected := local
			if route == "delegated" {
				selected = host
			}
			selected.failure = nil
			if status != 200 {
				selected.failure = fw.IdentityContractError("FIXTURE_ERROR", status)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/?framework_debug_route="+route, bytes.NewBufferString(`{"operation":"lookup","provider_subject":"shop:example:customer:1"}`))
			c.Request = c.Request.WithContext(middleware.ContextWithTenantUUID(c.Request.Context(), "11111111-1111-4111-8111-111111111111"))
			h.ExternalIdentities(c)
			require.Equal(t, status, rec.Code, rec.Body.String())
		}
	}
	require.Equal(t, 4, local.calls)
	require.Equal(t, 4, host.calls)
	for _, body := range []string{`{"operation":"lookup","tenant_uuid":"override"}`, `{"operation":"lookup","customer":{"display_name":"must not create"}}`} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/?framework_debug_route=local", bytes.NewBufferString(body))
		h.ExternalIdentities(c)
		require.Equal(t, 400, rec.Code)
	}
	// Headers/query do not supply trusted tenancy.
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/?framework_debug_route=local&tenant_uuid=spoof", bytes.NewBufferString(`{"operation":"lookup"}`))
	h.ExternalIdentities(c)
	require.Equal(t, 400, rec.Code)
	require.Equal(t, 4, local.calls)
}
