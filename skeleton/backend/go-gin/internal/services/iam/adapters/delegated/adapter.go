package delegated

import (
	"context"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
)

type delegatedProxy interface {
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
	GetDirectoryMember(ctx context.Context, memberUUID string) (*authproxy.DirectoryMember, error)
	BatchGetDirectoryMembers(ctx context.Context, memberUUIDs []string) ([]authproxy.DirectoryMember, error)
	BatchResolveDirectoryMembers(ctx context.Context, memberUUIDs []string) (*authproxy.DirectoryMemberResolution, error)
	BatchResolveDirectoryMembersByDisplayNames(ctx context.Context, displayNames []string) (*authproxy.DirectoryMemberDisplayNameResolution, error)
	ListDirectoryDepartments(ctx context.Context) ([]authproxy.DirectoryDepartment, error)
	ListDirectoryRoles(ctx context.Context) ([]authproxy.DirectoryRole, error)
	ListDirectoryPermissions(ctx context.Context) ([]authproxy.DirectoryPermission, error)
	CheckDirectoryAuthorization(ctx context.Context, request authproxy.DirectoryAuthorizationRequest) (*authproxy.AuthorizationDecision, error)
}

type identityResolver interface {
	ResolveIdentityContext(ctx context.Context, accessToken string) (*fwiamcontracts.IdentityContext, error)
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

func (a *Adapter) Authorize(ctx context.Context, req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	r, err := a.proxy.CheckDirectoryAuthorization(ctx, authproxy.DirectoryAuthorizationRequest{
		MemberUUID: req.MemberUUID,
		UserUUID:   req.UserUUID,
		Resource:   req.Resource,
		Action:     req.Action,
		TraceID:    req.TraceID,
	})
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if r == nil {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated authorization response is empty")
	}
	return &fwiamcontracts.AuthorizationDecision{Allowed: r.Allowed, ReasonCode: r.ReasonCode, Resource: req.Resource, Action: req.Action, TenantUUID: req.TenantUUID, UserUUID: req.UserUUID, MemberUUID: req.MemberUUID, Mode: string(fwiamcontracts.IAMAdapterModeDelegated), TraceID: req.TraceID}, nil
}

func (a *Adapter) ResolveIdentity(ctx context.Context, bearerToken string) (*fwiamcontracts.IdentityContext, error) {
	if resolver, ok := a.proxy.(identityResolver); ok {
		return resolver.ResolveIdentityContext(ctx, bearerToken)
	}
	me, err := a.proxy.MeContext(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	return authproxy.IdentityContextFromMeContext(me)
}
