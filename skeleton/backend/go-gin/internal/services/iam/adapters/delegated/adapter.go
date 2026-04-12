package delegated

import (
	"context"
	"strconv"
	"strings"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
)

type delegatedProxy interface {
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
}

// Adapter 将 delegated proxy 能力映射到 framework IAM 契约。
type Adapter struct {
	proxy delegatedProxy
}

// NewBundle 构造 delegated 模式的 framework IAM 绑定能力。
func NewBundle(proxy delegatedProxy) (fwiamadapters.Bundle, error) {
	if proxy == nil {
		return fwiamadapters.Bundle{}, fwiamerrors.New(fwiamerrors.CodeAdapterNotBound, "delegated auth proxy is nil")
	}
	adapter := &Adapter{proxy: proxy}
	return fwiamadapters.Bundle{
		Directory: adapter,
		Authz:     adapter,
		Context:   adapter,
	}, nil
}

func (a *Adapter) GetTenant(_ context.Context, tenantUUID string) (*fwiamcontracts.Tenant, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant uuid is required")
	}
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID}, nil
}

func (a *Adapter) ListDepartments(_ context.Context, _ string) ([]fwiamcontracts.Department, error) {
	return []fwiamcontracts.Department{}, nil
}

func (a *Adapter) ListMembers(_ context.Context, _ string) ([]fwiamcontracts.Member, error) {
	return []fwiamcontracts.Member{}, nil
}

func (a *Adapter) ListRoles(_ context.Context, _ string) ([]fwiamcontracts.Role, error) {
	return []fwiamcontracts.Role{}, nil
}

func (a *Adapter) ListPermissions(_ context.Context, _ string) ([]fwiamcontracts.Permission, error) {
	return []fwiamcontracts.Permission{}, nil
}

func (a *Adapter) Authorize(_ context.Context, req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	return &fwiamcontracts.AuthorizationDecision{
		Allowed:    false,
		ReasonCode: fwiamerrors.CodeUpstreamDependency,
		Resource:   req.Resource,
		Action:     req.Action,
		TenantUUID: req.TenantUUID,
		UserID:     req.UserID,
		Mode:       string(fwiamcontracts.IAMModeDelegated),
		TraceID:    req.TraceID,
	}, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated authorize mapping is not enabled yet")
}

func (a *Adapter) ResolveIdentity(ctx context.Context, bearerToken string) (*fwiamcontracts.IdentityContext, error) {
	me, err := a.proxy.MeContext(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	if me == nil {
		return nil, fwiamerrors.New(fwiamerrors.CodeUnauthorized, "empty delegated identity context")
	}
	identity := &fwiamcontracts.IdentityContext{
		TenantUUID: strings.TrimSpace(me.CurrentTenantUUID),
		TraceID:    "",
	}
	if len(me.Members) > 0 {
		member := me.Members[0]
		identity.MemberID = strconv.FormatUint(member.MemberID, 10)
	}
	return identity, nil
}
