package local

import (
	"context"
	"strings"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

type tokenContextResolver interface {
	UserContextFromToken(ctx context.Context, bearer string) (*iamservice.UserContext, error)
}

// Adapter 将 skeleton local IAM 能力映射为 framework IAM 契约。
type Adapter struct {
	directory iamservice.IAMDirectory
}

// NewBundle 构造 local 模式的 framework IAM 绑定能力。
func NewBundle(directory iamservice.IAMDirectory) (fwiamadapters.Bundle, error) {
	if directory == nil {
		return fwiamadapters.Bundle{}, fwiamerrors.New(fwiamerrors.CodeAdapterNotBound, "local iam directory is nil")
	}
	adapter := &Adapter{directory: directory}
	return fwiamadapters.Bundle{
		Directory: adapter,
		Authz:     adapter,
		Context:   adapter,
	}, nil
}

func (a *Adapter) Authorize(ctx context.Context, req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	tc := iamservice.TenantContext{
		TenantUUID: req.TenantUUID,
		UserUUID:   req.UserUUID,
		MemberUUID: req.MemberUUID,
	}
	err := a.directory.CheckPermission(ctx, tc, req.Resource, req.Action)
	if err == nil {
		return &fwiamcontracts.AuthorizationDecision{
			Allowed:    true,
			Resource:   req.Resource,
			Action:     req.Action,
			TenantUUID: req.TenantUUID,
			UserUUID:   req.UserUUID,
			MemberUUID: req.MemberUUID,
			Mode:       string(fwiamcontracts.IAMAdapterModeLocal),
			TraceID:    req.TraceID,
		}, nil
	}
	return &fwiamcontracts.AuthorizationDecision{
		Allowed:    false,
		ReasonCode: fwiamerrors.CodeForbidden,
		Resource:   req.Resource,
		Action:     req.Action,
		TenantUUID: req.TenantUUID,
		UserUUID:   req.UserUUID,
		MemberUUID: req.MemberUUID,
		Mode:       string(fwiamcontracts.IAMAdapterModeLocal),
		TraceID:    req.TraceID,
	}, nil
}

func (a *Adapter) ResolveIdentity(ctx context.Context, bearerToken string) (*fwiamcontracts.IdentityContext, error) {
	resolver, ok := a.directory.(tokenContextResolver)
	if !ok {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "local iam directory does not support token context resolution")
	}
	userCtx, err := resolver.UserContextFromToken(ctx, bearerToken)
	if err != nil {
		return nil, err
	}
	if userCtx == nil {
		return nil, fwiamerrors.New(fwiamerrors.CodeUnauthorized, "user context not found")
	}
	return &fwiamcontracts.IdentityContext{
		TenantUUID:  firstNonEmpty(userCtx.TenantUUID, userCtx.TenantUuid),
		UserUUID:    strings.TrimSpace(userCtx.UserUUID),
		MemberUUID:  strings.TrimSpace(userCtx.MemberUUID),
		Roles:       append([]string{}, userCtx.Roles...),
		Permissions: append([]string{}, userCtx.Permissions...),
		PolicyVer:   userCtx.PolicyVersion,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
