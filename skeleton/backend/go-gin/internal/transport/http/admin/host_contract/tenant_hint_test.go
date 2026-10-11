package host_contract

import (
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeAuthenticatedTenantHint(t *testing.T) {
	for _, tc := range []struct {
		name, tenant, header, query string
		duplicate                   bool
		want                        int
	}{
		{name: "browser", tenant: probeTenantUUID, header: probeTenantUUID, want: 200},
		{name: "no_hint", tenant: probeTenantUUID, want: 200},
		{name: "mismatch", tenant: probeTenantUUID, header: probeMemberUUID, want: 400},
		{name: "header_only", header: probeTenantUUID, want: 401},
		{name: "query_even_matching", tenant: probeTenantUUID, query: "?tenant_uuid=" + probeTenantUUID, want: 400},
		{name: "empty_query", tenant: probeTenantUUID, query: "?tenant_uuid=", want: 400},
		{name: "duplicate", tenant: probeTenantUUID, header: probeTenantUUID, duplicate: true, want: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := &directoryStub{}
			handler := NewHandler(&app.Deps{IAMDirectoryService: directory})
			req := httptest.NewRequest("POST", "/admin/host-contract/probe"+tc.query, strings.NewReader(`{"module":"iam","operation":"member.get","input":{"member_uuid":"`+probeMemberUUID+`"}}`))
			if tc.tenant != "" {
				req = req.WithContext(authx.ContextWithTenantUUID(req.Context(), tc.tenant))
			}
			if tc.header != "" {
				req.Header.Set("tenant_uuid", tc.header)
			}
			if tc.duplicate {
				req.Header.Add("tenant_uuid", tc.header)
			}
			response := executeProbeRequest(t, handler, req)
			if response.Code != tc.want {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.want == 200 && directory.tenant != probeTenantUUID {
				t.Fatal("credential tenant not used")
			}
			if tc.want != 200 && directory.tenant != "" {
				t.Fatal("rejected request reached service")
			}
		})
	}
}
