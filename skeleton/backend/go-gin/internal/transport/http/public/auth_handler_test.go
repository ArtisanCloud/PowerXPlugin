package public

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

func TestAuthHandler_LoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := &iamservice.AuthTokens{
		TokenType:    "Bearer",
		AccessToken:  "access",
		RefreshToken: "refresh",
		Scope:        "access",
		ExpiresIn:    3600,
		ExpiresAt:    time.Unix(1710000000, 0),
	}
	proxy := &stubProxy{
		loginFn: func(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error) {
			require.Equal(t, "user@example.com", req.Identifier)
			require.Equal(t, "secret", req.Password)
			return tokens, nil
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeDelegated, AuthProxy: proxy})
	router.POST("/auth/login", handler.Login)

	body := `{"identifier":"user@example.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var payload contracts.APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	data := payload.Data.(map[string]any)
	require.Equal(t, "access", data["access_token"])
	require.Equal(t, float64(3600), data["expires_in"])
	require.Equal(t, "refresh", data["refresh_token"])
}

func TestAuthHandler_LoginUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proxy := &stubProxy{
		loginFn: func(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error) {
			return nil, iamservice.ErrAuthUnavailable
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeDelegated, AuthProxy: proxy})
	router.POST("/auth/login", handler.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"identifier":"a","password":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	var payload contracts.APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.False(t, payload.Success)
}

func TestAuthHandler_RefreshUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proxy := &stubProxy{
		refreshFn: func(ctx context.Context, token string) (*iamservice.AuthTokens, error) {
			return nil, iamservice.ErrUnauthorized
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeDelegated, AuthProxy: proxy})
	router.POST("/auth/refresh", handler.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refresh_token":"abc"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthHandler_MeContextSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctxResp := &authproxy.MeContext{
		IsRoot:            true,
		CurrentTenantUUID: "tenant-42",
		CurrentMemberUUID: "member-uuid-7",
		Members: []authproxy.MeMemberBrief{{
			TenantUUID: "tenant-42",
			TenantName: "ACME",
			MemberID:   7,
			MemberUUID: "member-uuid-7",
			IsAdmin:    true,
		}},
	}
	proxy := &stubProxy{
		meFn: func(ctx context.Context, token string) (*authproxy.MeContext, error) {
			require.Equal(t, "bearer-token", token)
			return ctxResp, nil
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeDelegated, AuthProxy: proxy})
	router.GET("/auth/me/context", handler.MeContext)

	req := httptest.NewRequest(http.MethodGet, "/auth/me/context", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var payload contracts.APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	data := payload.Data.(map[string]any)
	require.Equal(t, "tenant-42", data["current_tenant_uuid"])
	require.Equal(t, "member-uuid-7", data["current_member_uuid"])
}

type stubProxy struct {
	loginFn   func(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, error)
	refreshFn func(context.Context, string) (*iamservice.AuthTokens, error)
	logoutFn  func(context.Context, string) error
	meFn      func(context.Context, string) (*authproxy.MeContext, error)
}

func (s *stubProxy) Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, req)
	}
	return nil, nil
}

func (s *stubProxy) Refresh(ctx context.Context, token string) (*iamservice.AuthTokens, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, token)
	}
	return nil, nil
}

func (s *stubProxy) Logout(ctx context.Context, token string) error {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, token)
	}
	return nil
}

func (s *stubProxy) MeContext(ctx context.Context, token string) (*authproxy.MeContext, error) {
	if s.meFn != nil {
		return s.meFn(ctx, token)
	}
	return nil, nil
}

func TestAuthHandler_LocalLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	local := &localDirStub{
		loginFn: func(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
			return &iamservice.AuthTokens{
				TokenType:    "Bearer",
				AccessToken:  "local-access",
				RefreshToken: "local-refresh",
				ExpiresIn:    600,
				Scope:        "access",
				ExpiresAt:    time.Now().Add(time.Minute),
			}, &iamservice.UserContext{TenantUUID: "tenant-1", TenantUuid: "1"}, nil
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeLocal, IAMDirectory: local})
	router.POST("/auth/login", handler.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"identifier":"admin","password":"pwd"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var payload contracts.APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	data := payload.Data.(map[string]any)
	require.Equal(t, "local-access", data["access_token"])
}

func TestAuthHandler_LocalMeContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	local := &localDirStub{
		ctxFn: func(ctx context.Context, token string) (*iamservice.UserContext, error) {
			return &iamservice.UserContext{
				TenantUUID: "tenant-local",
				TenantUuid: "9",
				TenantKey:  "00000000-0000-0000-0000-000000000001",
				TenantName: "Local",
				TenantID:   9,
				UserID:     5,
				UserUUID:   "user-uuid-5",
				Username:   "admin",
				MemberID:   12,
				MemberUUID: "member-uuid-12",
				IsRoot:     true,
				Roles:      []string{"system.admin"},
			}, nil
		},
	}
	router := gin.New()
	handler := NewAuthHandler(&app.Deps{IAMMode: iamservice.IAMModeLocal, IAMDirectory: local})
	router.GET("/auth/me/context", handler.MeContext)

	req := httptest.NewRequest(http.MethodGet, "/auth/me/context", nil)
	req.Header.Set("Authorization", "Bearer local-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var payload contracts.APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	data := payload.Data.(map[string]any)
	require.True(t, data["is_root"].(bool))
	require.Equal(t, "tenant-local", data["current_tenant_uuid"])
	require.EqualValues(t, 9, data["current_tenant_id"])
	require.EqualValues(t, 12, data["current_member_id"])
	require.Equal(t, "member-uuid-12", data["current_member_uuid"])
	user := data["user"].(map[string]any)
	require.EqualValues(t, 5, user["id"])
	require.Equal(t, "user-uuid-5", user["uuid"])
	require.Equal(t, "user-uuid-5", user["user_uuid"])
	tenant := data["tenant"].(map[string]any)
	require.Equal(t, "tenant-local", tenant["uuid"])
	members := data["members"].([]any)
	require.Len(t, members, 1)
	member := members[0].(map[string]any)
	require.Equal(t, "tenant-local", member["tenant_uuid"])
	require.EqualValues(t, 9, member["tenant_id"])
	require.Equal(t, "member-uuid-12", member["member_uuid"])
	require.Equal(t, true, member["is_admin"])
}

type localDirStub struct {
	loginFn   func(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error)
	refreshFn func(context.Context, string) (*iamservice.AuthTokens, error)
	logoutFn  func(context.Context, string) error
	ctxFn     func(context.Context, string) (*iamservice.UserContext, error)
}

func (l *localDirStub) Mode() iamservice.IAMMode { return iamservice.IAMModeLocal }

func (l *localDirStub) Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
	if l.loginFn != nil {
		return l.loginFn(ctx, req)
	}
	return nil, nil, nil
}

func (l *localDirStub) Refresh(ctx context.Context, token string) (*iamservice.AuthTokens, error) {
	if l.refreshFn != nil {
		return l.refreshFn(ctx, token)
	}
	return nil, nil
}

func (l *localDirStub) Logout(ctx context.Context, token string) error {
	if l.logoutFn != nil {
		return l.logoutFn(ctx, token)
	}
	return nil
}

func (l *localDirStub) CurrentUser(ctx context.Context) (*iamservice.UserContext, error) {
	return nil, nil
}

func (l *localDirStub) ListRoles(ctx context.Context, tenantUUID string) ([]iamservice.RoleInfo, error) {
	return nil, nil
}

func (l *localDirStub) ListDepartments(ctx context.Context, tenantUUID string) ([]iamservice.DepartmentInfo, error) {
	return nil, nil
}

func (l *localDirStub) CheckPermission(ctx context.Context, tc iamservice.TenantContext, resource, action string) error {
	return nil
}

func (l *localDirStub) UserContextFromToken(ctx context.Context, token string) (*iamservice.UserContext, error) {
	if l.ctxFn != nil {
		return l.ctxFn(ctx, token)
	}
	return nil, nil
}
