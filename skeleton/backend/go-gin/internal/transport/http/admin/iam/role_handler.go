package iam

import (
	"errors"
	"net/http"
	"strings"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service   *srviam.RoleService
	mode      srviam.IAMAdapterMode
	directory fwiamcontracts.DirectoryService
}

func NewRoleHandler(svc *srviam.RoleService, mode srviam.IAMAdapterMode, directories ...fwiamcontracts.DirectoryService) *RoleHandler {
	var directory fwiamcontracts.DirectoryService
	if len(directories) > 0 {
		directory = directories[0]
	}
	return &RoleHandler{service: svc, mode: mode, directory: directory}
}

func (h *RoleHandler) Get(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		tenantUUID := admincommon.ResolveTenantUUID(c)
		if strings.TrimSpace(tenantUUID) == "" {
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
			return
		}
		roles, ok := h.delegatedRoles(c, tenantUUID)
		if !ok {
			return
		}
		id := strings.TrimSpace(c.Param("role_uuid"))
		for _, role := range roles {
			if role.RoleUUID == id {
				contracts.ResponseSuccess(c, role)
				return
			}
		}
		contracts.ResponseNotFound(c, "role not found")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roleUUID := strings.TrimSpace(c.Param("role_uuid"))
	if !validUUIDs([]string{roleUUID}) {
		contracts.ResponseBadRequest(c, "invalid role_uuid")
		return
	}
	role, err := h.service.GetByUUID(c.Request.Context(), roleUUID)
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

func (h *RoleHandler) List(c *gin.Context) {
	type roleListQueryInput struct {
		TenantUUID string `form:"tenant_uuid"`
		Query      string `form:"q"`
		ScopeType  string `form:"scope_type"`
	}
	var query roleListQueryInput
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	if strings.TrimSpace(query.TenantUUID) == "" {
		query.TenantUUID = admincommon.ResolveTenantUUID(c)
	}
	if strings.TrimSpace(query.TenantUUID) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}
	if h.mode == srviam.IAMAdapterModeDelegated {
		roles, ok := h.delegatedRoles(c, query.TenantUUID)
		if !ok {
			return
		}
		contracts.ResponseSuccess(c, gin.H{"items": roles})
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roles, err := h.service.List(c.Request.Context(), srviam.RoleFilter{
		TenantUUID: query.TenantUUID,
		Query:      query.Query,
		ScopeType:  query.ScopeType,
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": roles})
}

func (h *RoleHandler) Create(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "role write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		req.TenantUUID = admincommon.ResolveTenantUUID(c)
	}
	if (req.CloneRoleUUID != nil && !validUUIDs([]string{*req.CloneRoleUUID})) || !validUUIDs(req.PermissionUUIDs) || !validUUIDs(req.MemberUUIDs) {
		contracts.ResponseBadRequest(c, "role relationship fields must contain UUID values")
		return
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}
	actorID := currentActorID(c)
	role, err := h.service.CreateByUUID(c.Request.Context(), srviam.CreateRoleUUIDInput{
		TenantUUID:      req.TenantUUID,
		Code:            req.Code,
		Name:            req.Name,
		Description:     req.Description,
		ScopeType:       req.ScopeType,
		CloneRoleUUID:   req.CloneRoleUUID,
		PermissionUUIDs: req.PermissionUUIDs,
		MemberUUIDs:     req.MemberUUIDs,
		ActorID:         actorID,
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseCreated(c, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "role write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roleUUID := strings.TrimSpace(c.Param("role_uuid"))
	if !validUUIDs([]string{roleUUID}) {
		contracts.ResponseBadRequest(c, "invalid role_uuid")
		return
	}
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	role, err := h.service.UpdateByUUID(c.Request.Context(), roleUUID, srviam.UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		ScopeType:   req.ScopeType,
		ActorID:     currentActorID(c),
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "role write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roleUUID := strings.TrimSpace(c.Param("role_uuid"))
	if !validUUIDs([]string{roleUUID}) {
		contracts.ResponseBadRequest(c, "invalid role_uuid")
		return
	}
	if handleRoleError(c, h.service.DeleteByUUID(c.Request.Context(), roleUUID, currentActorID(c))) {
		return
	}
	contracts.ResponseSuccessWithMessage(c, gin.H{"role_uuid": roleUUID}, "deleted")
}

func (h *RoleHandler) delegatedRoles(c *gin.Context, tenantUUID string) ([]fwiamcontracts.Role, bool) {
	if h.directory == nil {
		contracts.ResponseServiceUnavailable(c, "IAM delegated directory provider is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "delegated"})
		return nil, false
	}
	roles, err := h.directory.ListRoles(c.Request.Context(), tenantUUID)
	if err != nil {
		respondIAMError(c, err)
		return nil, false
	}
	return roles, true
}

type RolePermissionsHandler struct {
	service *srviam.RoleService
	mode    srviam.IAMAdapterMode
}

func NewRolePermissionsHandler(svc *srviam.RoleService, mode srviam.IAMAdapterMode) *RolePermissionsHandler {
	return &RolePermissionsHandler{service: svc, mode: mode}
}

func (h *RolePermissionsHandler) Replace(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "role permission write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roleUUID := strings.TrimSpace(c.Param("role_uuid"))
	if !validUUIDs([]string{roleUUID}) {
		contracts.ResponseBadRequest(c, "invalid role_uuid")
		return
	}
	var req ReplaceRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		req.TenantUUID = admincommon.ResolveTenantUUID(c)
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}
	if !validUUIDs(req.PermissionUUIDs) {
		contracts.ResponseBadRequest(c, "permission_uuids must contain UUID values")
		return
	}
	role, err := h.service.ReplacePermissionsByUUID(c.Request.Context(), srviam.ReplaceRolePermissionsUUIDInput{
		RoleUUID:        roleUUID,
		TenantUUID:      req.TenantUUID,
		PermissionUUIDs: req.PermissionUUIDs,
		ActorID:         currentActorID(c),
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

type RoleMembersHandler struct {
	service *srviam.RoleService
	mode    srviam.IAMAdapterMode
}

func NewRoleMembersHandler(svc *srviam.RoleService, mode srviam.IAMAdapterMode) *RoleMembersHandler {
	return &RoleMembersHandler{service: svc, mode: mode}
}

func (h *RoleMembersHandler) Add(c *gin.Context) {
	h.mutateMembers(c, true)
}

func (h *RoleMembersHandler) Remove(c *gin.Context) {
	h.mutateMembers(c, false)
}

func (h *RoleMembersHandler) mutateMembers(c *gin.Context, add bool) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "role member write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	roleUUID := strings.TrimSpace(c.Param("role_uuid"))
	if !validUUIDs([]string{roleUUID}) {
		contracts.ResponseBadRequest(c, "invalid role_uuid")
		return
	}
	var req RoleMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		req.TenantUUID = admincommon.ResolveTenantUUID(c)
	}
	if strings.TrimSpace(req.TenantUUID) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}
	if !validUUIDs(req.MemberUUIDs) {
		contracts.ResponseBadRequest(c, "member_uuids must contain UUID values")
		return
	}
	input := srviam.RoleMembersUUIDInput{
		RoleUUID:    roleUUID,
		TenantUUID:  req.TenantUUID,
		MemberUUIDs: req.MemberUUIDs,
		ActorID:     currentActorID(c),
	}
	var err error
	if add {
		err = h.service.AddMembersByUUID(c.Request.Context(), input)
	} else {
		err = h.service.RemoveMembersByUUID(c.Request.Context(), input)
	}
	if handleRoleError(c, err) {
		return
	}
	action := "removed"
	if add {
		action = "added"
	}
	contracts.ResponseSuccessWithMessage(c, gin.H{"role_uuid": roleUUID, "member_uuids": req.MemberUUIDs}, action)
}

type PermissionHandler struct {
	service   *srviam.RoleService
	mode      srviam.IAMAdapterMode
	directory fwiamcontracts.DirectoryService
}

func NewPermissionHandler(svc *srviam.RoleService, mode srviam.IAMAdapterMode, directories ...fwiamcontracts.DirectoryService) *PermissionHandler {
	var directory fwiamcontracts.DirectoryService
	if len(directories) > 0 {
		directory = directories[0]
	}
	return &PermissionHandler{service: svc, mode: mode, directory: directory}
}

func (h *PermissionHandler) List(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		tenantUUID := admincommon.ResolveTenantUUID(c)
		if strings.TrimSpace(tenantUUID) == "" {
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
			return
		}
		if h.directory == nil {
			contracts.ResponseServiceUnavailable(c, "IAM delegated directory provider is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "delegated"})
			return
		}
		perms, err := h.directory.ListPermissions(c.Request.Context(), tenantUUID)
		if err != nil {
			respondIAMError(c, err)
			return
		}
		contracts.ResponseSuccess(c, gin.H{"items": perms})
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local role service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	perms, err := h.service.ListPermissions(c.Request.Context())
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": perms})
}

func currentActorID(c *gin.Context) *uint64 {
	tc, _ := authmw.GetTenantContext(c)
	if tc.UserID <= 0 {
		return nil
	}
	uid := uint64(tc.UserID)
	return &uid
}

func handleRoleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, srviam.ErrTenantRequired):
		contracts.ResponseBadRequest(c, err.Error())
	case errors.Is(err, srviam.ErrPermissionDenied):
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodePermissionDenied, err.Error())
	case errors.Is(err, srviam.ErrRoleNotFound):
		contracts.ResponseNotFound(c, err.Error())
	default:
		contracts.ResponseInternalError(c, err)
	}
	return true
}
