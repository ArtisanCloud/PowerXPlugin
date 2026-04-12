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
	if resolver, ok := a.proxy.(identityResolver); ok {
		return resolver.ResolveIdentityContext(ctx, bearerToken)
	}
	me, err := a.proxy.MeContext(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	return authproxy.IdentityContextFromMeContext(me)
}
