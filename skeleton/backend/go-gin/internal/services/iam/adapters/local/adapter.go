package local

import (
	"context"
	"strconv"
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

func (a *Adapter) GetTenant(_ context.Context, tenantUUID string) (*fwiamcontracts.Tenant, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant uuid is required")
	}
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID}, nil
}

func (a *Adapter) ListDepartments(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Department, error) {
	items, err := a.directory.ListDepartments(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	result := make([]fwiamcontracts.Department, 0, len(items))
	for _, item := range items {
		parentID := ""
		if item.ParentID != nil {
			parentID = strings.TrimSpace(toStringUint(*item.ParentID))
		}
		result = append(result, fwiamcontracts.Department{
			ID:         toStringUint(item.ID),
			TenantUUID: firstNonEmpty(item.TenantUUID, item.TenantUuid),
			Name:       item.Name,
			Code:       item.Code,
			ParentID:   parentID,
		})
	}
	return result, nil
}

func (a *Adapter) ListMembers(_ context.Context, _ string) ([]fwiamcontracts.Member, error) {
	// Phase 3 先完成模式接入；成员目录统一映射在 US2 细化。
	return []fwiamcontracts.Member{}, nil
}

func (a *Adapter) ListRoles(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Role, error) {
	items, err := a.directory.ListRoles(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	result := make([]fwiamcontracts.Role, 0, len(items))
	for _, item := range items {
		result = append(result, fwiamcontracts.Role{
			ID:          toStringUint(item.ID),
			TenantUUID:  firstNonEmpty(item.TenantUUID, item.TenantUuid),
			Code:        item.Code,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return result, nil
}

func (a *Adapter) ListPermissions(_ context.Context, _ string) ([]fwiamcontracts.Permission, error) {
	// Phase 3 先完成模式接入；权限目录统一映射在 US2 细化。
	return []fwiamcontracts.Permission{}, nil
}

func (a *Adapter) Authorize(ctx context.Context, req fwiamcontracts.AuthorizationRequest) (*fwiamcontracts.AuthorizationDecision, error) {
	tc := iamservice.TenantContext{
		TenantUUID: req.TenantUUID,
		UserID:     toUint64(req.UserID),
	}
	err := a.directory.CheckPermission(ctx, tc, req.Resource, req.Action)
	if err == nil {
		return &fwiamcontracts.AuthorizationDecision{
			Allowed:    true,
			Resource:   req.Resource,
			Action:     req.Action,
			TenantUUID: req.TenantUUID,
			UserID:     req.UserID,
			Mode:       string(fwiamcontracts.IAMModeLocal),
			TraceID:    req.TraceID,
		}, nil
	}
	return &fwiamcontracts.AuthorizationDecision{
		Allowed:    false,
		ReasonCode: fwiamerrors.CodeForbidden,
		Resource:   req.Resource,
		Action:     req.Action,
		TenantUUID: req.TenantUUID,
		UserID:     req.UserID,
		Mode:       string(fwiamcontracts.IAMModeLocal),
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
		UserID:      toStringUint(userCtx.UserID),
		MemberID:    toStringUint(userCtx.MemberID),
		Roles:       append([]string{}, userCtx.Roles...),
		Permissions: append([]string{}, userCtx.Permissions...),
		PolicyVer:   userCtx.PolicyVersion,
	}, nil
}

func toStringUint(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func toUint64(raw string) uint64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
