package middleware

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"time"
)

type identityStub struct {
	value contracts.DelegatedIdentity
	calls int
}

func (s *identityStub) ResolveDelegatedIdentity(context.Context, contracts.DelegatedIdentityQuery) (*contracts.DelegatedIdentity, error) {
	s.calls++
	v := s.value
	return &v, nil
}
func identityFixture() contracts.DelegatedIdentity {
	member := "33333333-3333-4333-8333-333333333333"
	return contracts.DelegatedIdentity{Schema: contracts.IdentityResultSchema, Mode: "delegated", PluginID: "com.example.identity", TenantUUID: "11111111-1111-4111-8111-111111111111", UserUUID: "22222222-2222-4222-8222-222222222222", MemberUUID: &member, Roles: []string{}, Permissions: []string{"records:read"}, PolicyVersion: "policy", PermsHash: "hash", SourceMethod: "GET", SourcePath: "/admin/records", SourceRequiredPermission: "records:read", Transport: "http", SourceRequestID: "source-request", SourceTraceID: "source-trace", TraceID: "resolve-trace", ProofExpiresAt: time.Now().UTC().Add(time.Minute), UserSessionExpiresAt: time.Now().UTC().Add(time.Hour)}
}
func TestIdentityMiddlewareChecksAuthorityBeforeBusiness(t *testing.T) {
	for _, row := range []struct {
		name   string
		status int
		modify func(*identityStub)
		ticket string
	}{
		{"member", 200, nil, "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"root-null-member", 200, func(s *identityStub) {
			s.value.IsRoot = true
			s.value.MemberUUID = nil
			s.value.Permissions = []string{}
		}, "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"missing-proof", 401, nil, ""},
		{"wrong-source-path", 403, func(s *identityStub) { s.value.SourcePath = "/admin/another" }, "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"wrong-tenant", 403, func(s *identityStub) { s.value.TenantUUID = "44444444-4444-4444-8444-444444444444" }, "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"role-name-does-not-grant", 403, func(s *identityStub) { s.value.Roles = []string{"super_admin"}; s.value.Permissions = []string{} }, "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	} {
		t.Run(row.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			s := &identityStub{value: identityFixture()}
			if row.modify != nil {
				row.modify(s)
			}
			committed := false
			engine := gin.New()
			engine.Use(RefreshDelegatedIdentity(DelegatedIdentityOptions{Resolver: s, PluginID: "com.example.identity", APIPrefix: "/api/v1", TenantAssertion: func(*gin.Context) string { return "11111111-1111-4111-8111-111111111111" }, Commit: func(*gin.Context, *contracts.DelegatedIdentity) { committed = true }}))
			engine.Use(RequireIdentityPermission(func(*gin.Context) string { return "records:read" }))
			engine.GET("/api/v1/admin/records", func(c *gin.Context) { c.Status(200) })
			req := httptest.NewRequest("GET", "/api/v1/admin/records", nil)
			req.Header.Set(contracts.IdentityContextHeader, row.ticket)
			req.Header.Set("Authorization", "Bearer inbound-plugin-token")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != row.status {
				t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
			}
			if row.ticket == "" && (s.calls != 0 || committed) {
				t.Fatal("missing proof reached resolver or context commit")
			}
		})
	}
}
