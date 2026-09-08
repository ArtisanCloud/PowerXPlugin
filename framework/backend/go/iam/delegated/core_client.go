package delegated

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// TokenProvider obtains the plugin's short-lived STS credential.
type TokenProvider interface {
	Token(context.Context) (string, error)
}
type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type CoreClientConfig struct {
	BaseURL string
	Tokens  TokenProvider
	Timeout time.Duration
	Client  *http.Client
}

// CoreClient is the official Framework STS transport for the IAM directory
// and authorization Host Contracts. Tenant scope is never serialised: Core
// derives it from the service credential.
type CoreClient struct {
	baseURL string
	tokens  TokenProvider
	timeout time.Duration
	client  *http.Client
}

func NewCoreClient(cfg CoreClientConfig) (*CoreClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("delegated IAM: base URL is required")
	}
	if cfg.Tokens == nil {
		return nil, errors.New("delegated IAM: STS token provider is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 500*time.Millisecond}
	}
	return &CoreClient{baseURL: apiBase(cfg.BaseURL), tokens: cfg.Tokens, timeout: timeout, client: client}, nil
}

func (c *CoreClient) GetTenant(ctx context.Context, asserted string) (*contracts.Tenant, error) {
	var out contracts.Tenant
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/tenant", nil, &out, ""); err != nil {
		return nil, err
	}
	if strings.TrimSpace(asserted) != "" && strings.TrimSpace(asserted) != out.TenantUUID {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated tenant response does not match assertion")
	}
	return &out, nil
}
func (c *CoreClient) ListDepartments(ctx context.Context, tenant string) ([]contracts.Department, error) {
	var out struct {
		Items []contracts.Department `json:"items"`
	}
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/departments", nil, &out, ""); err != nil {
		return nil, err
	}
	return checkTenant(out.Items, tenant, func(v contracts.Department) string { return v.TenantUUID })
}
func (c *CoreClient) ListMembers(ctx context.Context, tenant string) ([]contracts.Member, error) {
	page, err := c.ListMembersPage(ctx, tenant, contracts.MemberPageRequest{Page: 1, PageSize: 200})
	if err != nil {
		return nil, err
	}
	result := append([]contracts.Member(nil), page.Items...)
	for p := 2; int64(len(result)) < page.Total; p++ {
		next, e := c.ListMembersPage(ctx, tenant, contracts.MemberPageRequest{Page: p, PageSize: 200})
		if e != nil {
			return nil, e
		}
		if len(next.Items) == 0 {
			return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated member pagination is inconsistent")
		}
		result = append(result, next.Items...)
	}
	return result, nil
}
func (c *CoreClient) ListMembersPage(ctx context.Context, tenant string, req contracts.MemberPageRequest) (*contracts.MemberPage, error) {
	if req.Page < 1 || req.PageSize < 1 || req.PageSize > 200 {
		return nil, iamerrors.New(iamerrors.CodeInvalidArgument, "invalid member pagination")
	}
	path := "/tenant/iam/members?page=" + strconv.Itoa(req.Page) + "&page_size=" + strconv.Itoa(req.PageSize)
	var out struct {
		Items      []contracts.Member `json:"items"`
		Pagination struct {
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Total    int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := c.call(ctx, http.MethodGet, path, nil, &out, ""); err != nil {
		return nil, err
	}
	items, err := checkTenant(out.Items, tenant, func(v contracts.Member) string { return v.TenantUUID })
	if err != nil {
		return nil, err
	}
	return &contracts.MemberPage{Items: items, Page: out.Pagination.Page, PageSize: out.Pagination.PageSize, Total: out.Pagination.Total}, nil
}
func (c *CoreClient) GetMember(ctx context.Context, tenant, member string) (*contracts.Member, error) {
	if strings.TrimSpace(member) == "" {
		return nil, iamerrors.New(iamerrors.CodeInvalidArgument, "member_uuid is required")
	}
	var out contracts.Member
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/members/"+url.PathEscape(strings.TrimSpace(member)), nil, &out, ""); err != nil {
		return nil, err
	}
	if tenant != "" && out.TenantUUID != tenant {
		return nil, iamerrors.New(iamerrors.CodeMemberNotFound, "member not found")
	}
	return &out, nil
}
func (c *CoreClient) BatchGetMembers(ctx context.Context, tenant string, ids []string) ([]contracts.Member, error) {
	var out struct {
		Items []contracts.Member `json:"items"`
	}
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-get", map[string]any{"member_uuids": ids}, &out, ""); err != nil {
		return nil, err
	}
	return checkTenant(out.Items, tenant, func(v contracts.Member) string { return v.TenantUUID })
}
func (c *CoreClient) BatchResolveMembers(ctx context.Context, tenant string, ids []string) (*contracts.MemberResolution, error) {
	var out contracts.MemberResolution
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-resolve", map[string]any{"member_uuids": ids}, &out, ""); err != nil {
		return nil, err
	}
	items, err := checkTenant(out.Items, tenant, func(v contracts.Member) string { return v.TenantUUID })
	if err != nil {
		return nil, err
	}
	out.Items = items
	return &out, nil
}
func (c *CoreClient) BatchResolveMembersByDisplayNames(ctx context.Context, tenant string, names []string) (*contracts.MemberDisplayNameResolution, error) {
	var out contracts.MemberDisplayNameResolution
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-find-by-display-names", map[string]any{"display_names": names}, &out, ""); err != nil {
		return nil, err
	}
	for _, item := range out.Items {
		if item.Member != nil && tenant != "" && item.Member.TenantUUID != "" && item.Member.TenantUUID != tenant {
			return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated display-name response crosses tenant")
		}
	}
	return &out, nil
}
func (c *CoreClient) ListRoles(ctx context.Context, tenant string) ([]contracts.Role, error) {
	var out struct {
		Items []contracts.Role `json:"items"`
	}
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/roles", nil, &out, ""); err != nil {
		return nil, err
	}
	return checkTenant(out.Items, tenant, func(v contracts.Role) string { return v.TenantUUID })
}
func (c *CoreClient) ListPermissions(ctx context.Context, _ string) ([]contracts.Permission, error) {
	var out struct {
		Items []contracts.Permission `json:"items"`
	}
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/permissions", nil, &out, ""); err != nil {
		return nil, err
	}
	return out.Items, nil
}
func (c *CoreClient) Authorize(ctx context.Context, in contracts.AuthorizationRequest) (*contracts.AuthorizationDecision, error) {
	var out contracts.AuthorizationDecision
	body := map[string]any{"member_uuid": in.MemberUUID, "user_uuid": in.UserUUID, "resource": in.Resource, "action": in.Action, "trace_id": in.TraceID}
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/authorization:check", body, &out, ""); err != nil {
		return nil, err
	}
	out.Mode = string(contracts.IAMAdapterModeDelegated)
	return &out, nil
}
func (c *CoreClient) ResolveIdentity(ctx context.Context, bearer string) (*contracts.IdentityContext, error) {
	var out contracts.IdentityContext
	if strings.TrimSpace(bearer) == "" {
		return nil, iamerrors.New(iamerrors.CodeUnauthorized, "bearer token is required")
	}
	if err := c.call(ctx, http.MethodGet, "/admin/user/auth/me/context", nil, &out, bearer); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CoreClient) call(ctx context.Context, method, path string, body any, out any, bearer string) error {
	if c == nil || c.client == nil || c.tokens == nil {
		return iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated IAM client unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var r io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return iamerrors.Wrap(iamerrors.CodeInvalidArgument, "delegated IAM request invalid", err)
		}
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return iamerrors.Wrap(iamerrors.CodeUpstreamDependency, "delegated IAM request unavailable", err)
	}
	if strings.TrimSpace(bearer) == "" {
		token, e := c.tokens.Token(ctx)
		if e != nil || strings.TrimSpace(token) == "" {
			return iamerrors.Wrap(iamerrors.CodeUpstreamDependency, "delegated IAM STS unavailable", e)
		}
		bearer = token
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return iamerrors.Wrap(iamerrors.CodeUpstreamDependency, "delegated IAM request unavailable", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapCoreError(resp.StatusCode, raw)
	}
	return decodeCoreEnvelope(raw, out)
}
func apiBase(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if !strings.Contains(strings.ToLower(base), "/api/") {
		base += "/api/v1"
	}
	return base
}
func decodeCoreEnvelope(raw []byte, out any) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return iamerrors.Wrap(iamerrors.CodeUpstreamDependency, "delegated IAM response invalid", err)
	}
	if len(envelope.Data) == 0 {
		envelope.Data = raw
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return iamerrors.Wrap(iamerrors.CodeUpstreamDependency, "delegated IAM response invalid", err)
	}
	return nil
}
func mapCoreError(status int, raw []byte) error {
	var e struct {
		ReasonCode string `json:"reason_code"`
		Error      struct {
			ReasonCode string `json:"reason_code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	reason := e.ReasonCode
	if reason == "" {
		reason = e.Error.ReasonCode
	}
	switch reason {
	case "IAM_INVALID_ARGUMENT":
		return iamerrors.New(iamerrors.CodeInvalidArgument, "delegated IAM request invalid")
	case "IAM_UNAUTHORIZED":
		return iamerrors.New(iamerrors.CodeUnauthorized, "delegated IAM unauthorized")
	case "IAM_FORBIDDEN":
		return iamerrors.New(iamerrors.CodeForbidden, "delegated IAM forbidden")
	case "IAM_MEMBER_NOT_FOUND":
		return iamerrors.New(iamerrors.CodeMemberNotFound, "member not found")
	}
	if status == 400 {
		return iamerrors.New(iamerrors.CodeInvalidArgument, "delegated IAM request invalid")
	}
	if status == 401 {
		return iamerrors.New(iamerrors.CodeUnauthorized, "delegated IAM unauthorized")
	}
	if status == 403 {
		return iamerrors.New(iamerrors.CodeForbidden, "delegated IAM forbidden")
	}
	if status == 404 {
		return iamerrors.New(iamerrors.CodeMemberNotFound, "member not found")
	}
	return iamerrors.New(iamerrors.CodeUpstreamDependency, fmt.Sprintf("delegated IAM dependency failed: %d", status))
}
func checkTenant[T any](items []T, tenant string, get func(T) string) ([]T, error) {
	if strings.TrimSpace(tenant) == "" {
		return items, nil
	}
	for _, item := range items {
		if strings.TrimSpace(get(item)) != strings.TrimSpace(tenant) {
			return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated IAM response crosses tenant")
		}
	}
	return items, nil
}

var _ Host = (*CoreClient)(nil)
