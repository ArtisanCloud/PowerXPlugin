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
	"github.com/google/uuid"
)

// TokenProvider obtains the plugin's short-lived STS credential.
type TokenProvider interface {
	Token(context.Context) (string, error)
}
type TokenProviderFunc func(context.Context) (string, error)

func (f TokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type CoreClientConfig struct {
	PluginID string
	BaseURL  string
	Tokens   TokenProvider
	Timeout  time.Duration
	Client   *http.Client
}

// CoreClient is the official Framework STS transport for the IAM directory
// and authorization Host Contracts. Tenant scope is never serialised: Core
// derives it from the service credential.
type CoreClient struct {
	pluginID   string
	baseURL    string
	tokens     TokenProvider
	timeout    time.Duration
	client     *http.Client
	authScheme string
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
	return &CoreClient{pluginID: strings.TrimSpace(cfg.PluginID), baseURL: apiBase(cfg.BaseURL), tokens: cfg.Tokens, timeout: timeout, client: client, authScheme: "Bearer"}, nil
}

// NewCoreClientWithAPIKey uses a server-owned credential for Local + proxy.
// The caller's browser token is never forwarded to the Core IAM directory.
func NewCoreClientWithAPIKey(cfg CoreClientConfig, apiKey string) (*CoreClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("iam.api_key_required")
	}
	cfg.Tokens = TokenProviderFunc(func(context.Context) (string, error) { return apiKey, nil })
	client, err := NewCoreClient(cfg)
	if err != nil {
		return nil, err
	}
	client.authScheme = "ApiKey"
	return client, nil
}

// CoreRequestError retains the actual denial status and Core trace for diagnostics.
type CoreRequestError struct {
	Status  int
	TraceID string
	Err     error
}

func (e *CoreRequestError) Error() string { return e.Err.Error() }
func (e *CoreRequestError) Unwrap() error { return e.Err }

// coreTenant is Core's directory DTO. Its status is numeric and its business
// key is named key; neither field has the Framework model's wire shape.
type coreTenant struct {
	TenantUUID string `json:"tenant_uuid"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Status     int16  `json:"status"`
	Type       string `json:"type"`
	Plan       string `json:"plan"`
}

func (t coreTenant) tenant() contracts.Tenant {
	return contracts.Tenant{TenantUUID: t.TenantUUID, TenantKey: t.Key, Name: t.Name, Status: strconv.FormatInt(int64(t.Status), 10)}
}

// readTenant is shared by the public directory method and credential-scope
// validation. A supplied credential stays pinned across a multi-read operation.
func (c *CoreClient) readTenant(ctx context.Context, credential string) (*contracts.Tenant, error) {
	var out coreTenant
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/tenant", nil, &out, credential); err != nil {
		return nil, err
	}
	result := out.tenant()
	return &result, nil
}

func (c *CoreClient) GetTenant(ctx context.Context, asserted string) (*contracts.Tenant, error) {
	out, err := c.readTenant(ctx, "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(asserted) != "" && strings.TrimSpace(asserted) != out.TenantUUID {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, "delegated tenant response does not match assertion")
	}
	return out, nil
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

// coreMember is the numeric-status DTO published by Core's member directory.
// Keep the wire contract separate from the Framework's string-status model.
type coreMember struct {
	MemberUUID  string `json:"member_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
	UserUUID    string `json:"user_uuid"`
	DisplayName string `json:"display_name,omitempty"`
	Status      int16  `json:"status"`
}

func (m coreMember) member() contracts.Member {
	return contracts.Member{MemberUUID: m.MemberUUID, TenantUUID: m.TenantUUID, UserUUID: m.UserUUID, DisplayName: m.DisplayName, Status: strconv.FormatInt(int64(m.Status), 10)}
}

func coreMembers(items []coreMember, tenant string) ([]contracts.Member, error) {
	out := make([]contracts.Member, 0, len(items))
	for _, item := range items {
		out = append(out, item.member())
	}
	return checkTenant(out, tenant, func(v contracts.Member) string { return v.TenantUUID })
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
		Items      []coreMember `json:"items"`
		Pagination struct {
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Total    int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := c.call(ctx, http.MethodGet, path, nil, &out, ""); err != nil {
		return nil, err
	}
	items, err := coreMembers(out.Items, tenant)
	if err != nil {
		return nil, err
	}
	return &contracts.MemberPage{Items: items, Page: out.Pagination.Page, PageSize: out.Pagination.PageSize, Total: out.Pagination.Total}, nil
}
func (c *CoreClient) GetMember(ctx context.Context, tenant, member string) (*contracts.Member, error) {
	if strings.TrimSpace(member) == "" {
		return nil, iamerrors.New(iamerrors.CodeInvalidArgument, "member_uuid is required")
	}
	var out coreMember
	if err := c.call(ctx, http.MethodGet, "/tenant/iam/members/"+url.PathEscape(strings.TrimSpace(member)), nil, &out, ""); err != nil {
		return nil, err
	}
	if tenant != "" && out.TenantUUID != tenant {
		return nil, iamerrors.New(iamerrors.CodeMemberNotFound, "member not found")
	}
	result := out.member()
	return &result, nil
}
func (c *CoreClient) BatchGetMembers(ctx context.Context, tenant string, ids []string) ([]contracts.Member, error) {
	var out struct {
		Items []coreMember `json:"items"`
	}
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-get", map[string]any{"member_uuids": ids}, &out, ""); err != nil {
		return nil, err
	}
	return coreMembers(out.Items, tenant)
}
func (c *CoreClient) BatchResolveMembers(ctx context.Context, tenant string, ids []string) (*contracts.MemberResolution, error) {
	// Core's tolerant resolver intentionally omits tenant and status. The tenant
	// is scoped by the service credential, not supplied in the request body.
	var out struct {
		Items []struct {
			MemberUUID  string `json:"member_uuid"`
			UserUUID    string `json:"user_uuid"`
			DisplayName string `json:"display_name,omitempty"`
		} `json:"items"`
		MissingMemberUUIDs []string `json:"missing_member_uuids"`
	}
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-resolve", map[string]any{"member_uuids": ids}, &out, ""); err != nil {
		return nil, err
	}
	result := &contracts.MemberResolution{Items: make([]contracts.Member, 0, len(out.Items)), MissingMemberUUIDs: out.MissingMemberUUIDs}
	for _, item := range out.Items {
		result.Items = append(result.Items, contracts.Member{MemberUUID: item.MemberUUID, TenantUUID: tenant, UserUUID: item.UserUUID, DisplayName: item.DisplayName})
	}
	return result, nil
}
func (c *CoreClient) BatchResolveMembersByDisplayNames(ctx context.Context, tenant string, names []string) (*contracts.MemberDisplayNameResolution, error) {
	asserted, err := uuid.Parse(strings.TrimSpace(tenant))
	if err != nil || asserted == uuid.Nil || len(names) == 0 || len(names) > 200 {
		return nil, iamerrors.New(iamerrors.CodeInvalidArgument, iamerrors.CodeInvalidArgument)
	}
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return nil, iamerrors.New(iamerrors.CodeInvalidArgument, iamerrors.CodeInvalidArgument)
		}
	}
	if c == nil || c.client == nil || c.tokens == nil {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	credential, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(credential) == "" {
		return nil, iamerrors.Wrap(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency, err)
	}
	// Core's name resolver deliberately omits tenant_uuid. Verify the caller's
	// assertion against the credential's tenant before binding that scope to
	// the shared Member model. Pin one credential for both reads; never label
	// a result using an unchecked request parameter or a decoded token claim.
	scope, err := c.readTenant(ctx, credential)
	if err != nil {
		return nil, err
	}
	scoped, err := uuid.Parse(strings.TrimSpace(scope.TenantUUID))
	if err != nil || scoped == uuid.Nil {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
	}
	if scoped != asserted {
		return nil, iamerrors.New(iamerrors.CodeForbidden, iamerrors.CodeForbidden)
	}
	var out contracts.MemberDisplayNameResolution
	if err := c.call(ctx, http.MethodPost, "/tenant/iam/members:batch-find-by-display-names", map[string]any{"display_names": names}, &out, credential); err != nil {
		return nil, err
	}
	if len(out.Items) != len(names) {
		return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
	}
	for i := range out.Items {
		item := &out.Items[i]
		if !strings.EqualFold(strings.TrimSpace(item.DisplayName), strings.TrimSpace(names[i])) {
			return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
		}
		switch item.Status {
		case contracts.MemberDisplayNameResolutionFound:
			if item.Member == nil || !strings.EqualFold(strings.TrimSpace(item.Member.DisplayName), strings.TrimSpace(item.DisplayName)) {
				return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
			}
			memberUUID, memberErr := uuid.Parse(strings.TrimSpace(item.Member.MemberUUID))
			userUUID, userErr := uuid.Parse(strings.TrimSpace(item.Member.UserUUID))
			if memberErr != nil || memberUUID == uuid.Nil || userErr != nil || userUUID == uuid.Nil {
				return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
			}
			if item.Member.TenantUUID != "" {
				returnedTenant, tenantErr := uuid.Parse(strings.TrimSpace(item.Member.TenantUUID))
				if tenantErr != nil || returnedTenant != scoped {
					return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
				}
			}
			item.Member.TenantUUID = scoped.String()
			item.Member.MemberUUID = memberUUID.String()
			item.Member.UserUUID = userUUID.String()
		case contracts.MemberDisplayNameResolutionNotFound, contracts.MemberDisplayNameResolutionAmbiguous:
			if item.Member != nil {
				return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
			}
		default:
			return nil, iamerrors.New(iamerrors.CodeUpstreamDependency, iamerrors.CodeUpstreamDependency)
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
	return nil, iamerrors.New(iamerrors.CodeIdentityMigrationRequired, iamerrors.CodeIdentityMigrationRequired)
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
	req.Header.Set("Authorization", c.authScheme+" "+strings.TrimSpace(bearer))
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
		return &CoreRequestError{Status: resp.StatusCode, TraceID: resp.Header.Get("X-Trace-Id"), Err: mapCoreError(resp.StatusCode, raw)}
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
