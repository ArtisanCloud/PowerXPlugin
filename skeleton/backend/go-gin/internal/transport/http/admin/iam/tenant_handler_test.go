package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type tenantTestDirectory struct {
	getTenant func(tenantUUID string) (*fwiamcontracts.Tenant, error)
}

func (d tenantTestDirectory) GetTenant(_ context.Context, tenantUUID string) (*fwiamcontracts.Tenant, error) {
	if d.getTenant != nil {
		return d.getTenant(tenantUUID)
	}
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID}, nil
}

func (tenantTestDirectory) ListDepartments(context.Context, string) ([]fwiamcontracts.Department, error) {
	return nil, nil
}

func (tenantTestDirectory) ListMembers(context.Context, string) ([]fwiamcontracts.Member, error) {
	return nil, nil
}

func (tenantTestDirectory) GetMember(context.Context, string, string) (*fwiamcontracts.Member, error) {
	return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "member not found")
}

func (tenantTestDirectory) BatchGetMembers(context.Context, string, []string) ([]fwiamcontracts.Member, error) {
	return nil, nil
}
func (tenantTestDirectory) BatchResolveMembers(context.Context, string, []string) (*fwiamcontracts.MemberResolution, error) {
	return &fwiamcontracts.MemberResolution{}, nil
}

func (tenantTestDirectory) ListRoles(context.Context, string) ([]fwiamcontracts.Role, error) {
	return nil, nil
}

func (tenantTestDirectory) ListPermissions(context.Context, string) ([]fwiamcontracts.Permission, error) {
	return nil, nil
}

type tenantTestAuthz struct {
	authorize func(req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error)
}

func (a tenantTestAuthz) Authorize(_ context.Context, req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	if a.authorize != nil {
		return a.authorize(req)
	}
	return &fwiamcontracts.AuthorizationDecision{Allowed: true}, nil
}

func TestTenantHandler_DelegatedErrorSemantics401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTenantHandler(
		nil,
		tenantTestDirectory{
			getTenant: func(string) (*fwiamcontracts.Tenant, error) {
				return nil, fwiamerrors.New(fwiamerrors.CodeUnauthorized, "unauthorized")
			},
		},
		tenantTestAuthz{},
		iamservice.IAMAdapterModeDelegated,
	)
	router.GET("/tenants", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/tenants?tenant_uuid=11111111-1111-1111-1111-111111111111", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assertIAMStatusCode(t, resp, http.StatusUnauthorized, fwiamerrors.CodeUnauthorized)
}

func TestTenantHandler_DelegatedErrorSemantics403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTenantHandler(
		nil,
		tenantTestDirectory{},
		tenantTestAuthz{
			authorize: func(req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
				return &fwiamcontracts.AuthorizationDecision{
					Allowed:    false,
					Resource:   req.Resource,
					Action:     req.Action,
					TenantUUID: req.TenantUUID,
				}, nil
			},
		},
		iamservice.IAMAdapterModeDelegated,
	)
	router.GET("/tenants", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/tenants?tenant_uuid=11111111-1111-1111-1111-111111111111", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assertIAMStatusCode(t, resp, http.StatusForbidden, contracts.ErrCodeForbidden)
}

func TestTenantHandler_DelegatedErrorSemantics424(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTenantHandler(
		nil,
		tenantTestDirectory{},
		tenantTestAuthz{
			authorize: func(fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
				return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "upstream unavailable")
			},
		},
		iamservice.IAMAdapterModeDelegated,
	)
	router.GET("/tenants", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/tenants?tenant_uuid=11111111-1111-1111-1111-111111111111", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assertIAMStatusCode(t, resp, http.StatusFailedDependency, fwiamerrors.CodeUpstreamDependency)
}

func assertIAMStatusCode(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	t.Helper()
	if resp.Code != expectedStatus {
		t.Fatalf("status mismatch, got=%d body=%s", resp.Code, resp.Body.String())
	}
	var envelope contracts.APIResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != expectedCode {
		t.Fatalf("error code mismatch, got=%#v", envelope.Error)
	}
}
