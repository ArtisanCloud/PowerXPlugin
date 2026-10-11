package delegated

import (
	"context"
	"errors"
	"strconv"
	"strings"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

func (a *Adapter) GetTenant(ctx context.Context, tenantUUID string) (*fwiamcontracts.Tenant, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant uuid is required")
	}
	tenant, err := a.proxy.GetDirectoryTenant(ctx)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if tenant == nil || strings.TrimSpace(tenant.TenantUUID) != tenantUUID {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated tenant directory response is invalid")
	}
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID, TenantKey: strings.TrimSpace(tenant.TenantKey), Name: strings.TrimSpace(tenant.Name), Status: strings.TrimSpace(tenant.Status)}, nil
}

func (a *Adapter) ListDepartments(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Department, error) {
	items, err := a.proxy.ListDirectoryDepartments(ctx)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	out := make([]fwiamcontracts.Department, 0, len(items))
	for _, v := range items {
		if strings.TrimSpace(v.TenantUUID) != strings.TrimSpace(tenantUUID) {
			return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "department not found")
		}
		out = append(out, fwiamcontracts.Department{DepartmentUUID: v.DepartmentUUID, TenantUUID: v.TenantUUID, Name: v.Name, Code: v.Code, ParentDepartmentUUID: v.ParentDepartmentUUID})
	}
	return out, nil
}

func (a *Adapter) ListMembers(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Member, error) {
	const pageSize = 200
	first, err := a.ListMembersPage(ctx, tenantUUID, fwiamcontracts.MemberPageRequest{Page: 1, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	result := append([]fwiamcontracts.Member(nil), first.Items...)
	for page := 2; int64(len(result)) < first.Total; page++ {
		next, err := a.ListMembersPage(ctx, tenantUUID, fwiamcontracts.MemberPageRequest{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		if next.Total != first.Total || len(next.Items) == 0 {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated member directory pagination is inconsistent")
		}
		result = append(result, next.Items...)
	}
	if int64(len(result)) != first.Total {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated member directory pagination is inconsistent")
	}
	return result, nil
}

func (a *Adapter) ListMembersPage(ctx context.Context, tenantUUID string, request fwiamcontracts.MemberPageRequest) (*fwiamcontracts.MemberPage, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" || request.Page < 1 || request.PageSize < 1 || request.PageSize > 200 {
		return nil, fwiamerrors.New(fwiamerrors.CodeInvalidArgument, "tenant_uuid and pagination are required")
	}
	response, err := a.proxy.ListDirectoryMembers(ctx, request.Page, request.PageSize)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if response == nil || response.Pagination.Page != request.Page || response.Pagination.PageSize != request.PageSize || response.Pagination.Total < 0 || int64(len(response.Items)) > response.Pagination.Total {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated member directory page is invalid")
	}
	items := make([]fwiamcontracts.Member, 0, len(response.Items))
	for _, member := range response.Items {
		if strings.TrimSpace(member.MemberUUID) == "" || strings.TrimSpace(member.UserUUID) == "" || strings.TrimSpace(member.TenantUUID) != tenantUUID {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated member directory page is invalid")
		}
		items = append(items, *memberFromHost(member))
	}
	return &fwiamcontracts.MemberPage{Items: items, Page: response.Pagination.Page, PageSize: response.Pagination.PageSize, Total: response.Pagination.Total}, nil
}

func (a *Adapter) GetMember(ctx context.Context, tenantUUID, memberUUID string) (*fwiamcontracts.Member, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	memberUUID = strings.TrimSpace(memberUUID)
	if tenantUUID == "" || memberUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant_uuid and member_uuid are required")
	}
	member, err := a.proxy.GetDirectoryMember(ctx, memberUUID)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if member == nil || strings.TrimSpace(member.MemberUUID) == "" || strings.TrimSpace(member.TenantUUID) != strings.TrimSpace(tenantUUID) {
		return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "member not found")
	}
	return memberFromHost(*member), nil
}

func (a *Adapter) BatchGetMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) ([]fwiamcontracts.Member, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant_uuid is required")
	}
	normalized, err := iamservice.NormalizeMemberUUIDs(memberUUIDs)
	if err != nil {
		return nil, fwiamerrors.Wrap(fwiamerrors.CodeInvalidArgument, "member_uuids are invalid", err)
	}
	members, err := a.proxy.BatchGetDirectoryMembers(ctx, normalized)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	byUUID := make(map[string]authproxy.DirectoryMember, len(members))
	for _, member := range members {
		if strings.TrimSpace(member.MemberUUID) == "" || strings.TrimSpace(member.TenantUUID) != strings.TrimSpace(tenantUUID) {
			return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "member not found")
		}
		byUUID[strings.TrimSpace(member.MemberUUID)] = member
	}
	result := make([]fwiamcontracts.Member, 0, len(normalized))
	for _, memberUUID := range normalized {
		member, ok := byUUID[memberUUID]
		if !ok {
			return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "member not found")
		}
		result = append(result, *memberFromHost(member))
	}
	return result, nil
}

func (a *Adapter) BatchResolveMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) (*fwiamcontracts.MemberResolution, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant_uuid is required")
	}
	normalized, err := iamservice.NormalizeMemberUUIDs(memberUUIDs)
	if err != nil {
		return nil, fwiamerrors.Wrap(fwiamerrors.CodeInvalidArgument, "member_uuids are invalid", err)
	}
	resolution, err := a.proxy.BatchResolveDirectoryMembers(ctx, normalized)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if resolution == nil {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response is empty")
	}
	requested := make(map[string]struct{}, len(normalized))
	for _, memberUUID := range normalized {
		requested[memberUUID] = struct{}{}
	}
	result := &fwiamcontracts.MemberResolution{
		Items:              make([]fwiamcontracts.Member, 0, len(resolution.Items)),
		MissingMemberUUIDs: make([]string, 0, len(resolution.MissingMemberUUIDs)),
	}
	seen := make(map[string]struct{}, len(normalized))
	for _, member := range resolution.Items {
		memberUUID := strings.TrimSpace(member.MemberUUID)
		if _, ok := requested[memberUUID]; !ok || strings.TrimSpace(member.TenantUUID) != tenantUUID {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response is invalid")
		}
		if _, duplicate := seen[memberUUID]; duplicate {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response contains duplicates")
		}
		seen[memberUUID] = struct{}{}
		result.Items = append(result.Items, *memberFromHost(member))
	}
	for _, memberUUID := range resolution.MissingMemberUUIDs {
		memberUUID = strings.TrimSpace(memberUUID)
		if _, ok := requested[memberUUID]; !ok {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response contains unknown uuid")
		}
		if _, duplicate := seen[memberUUID]; duplicate {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response overlaps items and missing UUIDs")
		}
		seen[memberUUID] = struct{}{}
		result.MissingMemberUUIDs = append(result.MissingMemberUUIDs, memberUUID)
	}
	if len(seen) != len(normalized) {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated directory resolve response omits requested UUIDs")
	}
	return result, nil
}

func (a *Adapter) BatchResolveMembersByDisplayNames(ctx context.Context, tenantUUID string, displayNames []string) (*fwiamcontracts.MemberDisplayNameResolution, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant_uuid is required")
	}
	normalized, err := iamservice.NormalizeMemberDisplayNames(displayNames)
	if err != nil {
		return nil, fwiamerrors.Wrap(fwiamerrors.CodeInvalidArgument, "display_names are invalid", err)
	}
	resolution, err := a.proxy.BatchResolveDirectoryMembersByDisplayNames(ctx, normalized)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	if resolution == nil || len(resolution.Items) != len(normalized) {
		return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated display-name directory response is invalid")
	}
	result := &fwiamcontracts.MemberDisplayNameResolution{Items: make([]fwiamcontracts.MemberDisplayNameResolutionItem, 0, len(normalized))}
	for index, item := range resolution.Items {
		if strings.TrimSpace(item.DisplayName) != normalized[index] {
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated display-name directory response does not preserve request order")
		}
		out := fwiamcontracts.MemberDisplayNameResolutionItem{DisplayName: normalized[index], Status: fwiamcontracts.MemberDisplayNameResolutionStatus(strings.TrimSpace(item.Status))}
		switch out.Status {
		case fwiamcontracts.MemberDisplayNameResolutionFound:
			if item.Member == nil || strings.TrimSpace(item.Member.MemberUUID) == "" || strings.TrimSpace(item.Member.UserUUID) == "" || strings.TrimSpace(item.Member.DisplayName) == "" {
				return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated display-name directory response is invalid")
			}
			out.Member = &fwiamcontracts.Member{
				MemberUUID:  strings.TrimSpace(item.Member.MemberUUID),
				TenantUUID:  tenantUUID,
				UserUUID:    strings.TrimSpace(item.Member.UserUUID),
				DisplayName: strings.TrimSpace(item.Member.DisplayName),
			}
		case fwiamcontracts.MemberDisplayNameResolutionNotFound, fwiamcontracts.MemberDisplayNameResolutionAmbiguous:
			if item.Member != nil {
				return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated display-name directory response exposes an unresolved member")
			}
		default:
			return nil, fwiamerrors.New(fwiamerrors.CodeUpstreamDependency, "delegated display-name directory response has an unknown status")
		}
		result.Items = append(result.Items, out)
	}
	return result, nil
}

func mapDirectoryError(err error) error {
	var proxyErr *authproxy.ProxyError
	if errors.As(err, &proxyErr) {
		switch proxyErr.Status {
		case 400:
			return fwiamerrors.Wrap(fwiamerrors.CodeInvalidArgument, "delegated directory request is invalid", err)
		case 404:
			return fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "member not found")
		case 401:
			return fwiamerrors.Wrap(fwiamerrors.CodeUnauthorized, "delegated directory request was rejected", err)
		case 403:
			return fwiamerrors.Wrap(fwiamerrors.CodeForbidden, "delegated directory request was forbidden", err)
		case 502, 503:
			return fwiamerrors.Wrap(fwiamerrors.CodeUpstreamDependency, "delegated directory dependency is unavailable", err)
		}
	}
	return fwiamerrors.Wrap(fwiamerrors.CodeUpstreamDependency, "delegated directory lookup failed", err)
}

func (a *Adapter) ListRoles(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Role, error) {
	items, err := a.proxy.ListDirectoryRoles(ctx)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	out := make([]fwiamcontracts.Role, 0, len(items))
	for _, v := range items {
		if strings.TrimSpace(v.TenantUUID) != strings.TrimSpace(tenantUUID) {
			return nil, fwiamerrors.New(fwiamerrors.CodeMemberNotFound, "role not found")
		}
		out = append(out, fwiamcontracts.Role{RoleUUID: v.RoleUUID, TenantUUID: v.TenantUUID, Code: v.Code, Name: v.Name, Description: v.Description})
	}
	return out, nil
}

func memberFromHost(member authproxy.DirectoryMember) *fwiamcontracts.Member {
	return &fwiamcontracts.Member{MemberUUID: strings.TrimSpace(member.MemberUUID), TenantUUID: strings.TrimSpace(member.TenantUUID), UserUUID: strings.TrimSpace(member.UserUUID), DisplayName: strings.TrimSpace(member.DisplayName), Status: strconv.FormatInt(int64(member.Status), 10)}
}

func (a *Adapter) ListPermissions(ctx context.Context, _ string) ([]fwiamcontracts.Permission, error) {
	items, err := a.proxy.ListDirectoryPermissions(ctx)
	if err != nil {
		return nil, mapDirectoryError(err)
	}
	out := make([]fwiamcontracts.Permission, 0, len(items))
	for _, v := range items {
		out = append(out, fwiamcontracts.Permission{PermissionUUID: v.PermissionUUID, Resource: v.Resource, Action: v.Action, Scope: v.Scope})
	}
	return out, nil
}
