package iam

import (
	"net/http"
	"strconv"
	"strings"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	service   *srviam.DepartmentService
	mode      srviam.IAMAdapterMode
	directory fwiamcontracts.DirectoryService
}

type departmentNode struct {
	iamm.Department
	Children []*departmentNode `json:"children,omitempty"`
}

func NewDepartmentHandler(svc *srviam.DepartmentService, mode srviam.IAMAdapterMode, directories ...fwiamcontracts.DirectoryService) *DepartmentHandler {
	var directory fwiamcontracts.DirectoryService
	if len(directories) > 0 {
		directory = directories[0]
	}
	return &DepartmentHandler{service: svc, mode: mode, directory: directory}
}

func (h *DepartmentHandler) List(c *gin.Context) {
	var query DepartmentListQuery
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
		deps, ok := h.delegatedDepartments(c, query.TenantUUID)
		if !ok {
			return
		}
		contracts.ResponseSuccess(c, gin.H{"items": deps})
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local department service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	deps, err := h.service.List(c.Request.Context(), srviam.DepartmentFilter{TenantUUID: query.TenantUUID})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": deps})
}

func (h *DepartmentHandler) Tree(c *gin.Context) {
	var query DepartmentListQuery
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
		deps, ok := h.delegatedDepartments(c, query.TenantUUID)
		if !ok {
			return
		}
		contracts.ResponseSuccess(c, buildDelegatedDepartmentTree(deps))
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local department service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	deps, err := h.service.List(c.Request.Context(), srviam.DepartmentFilter{TenantUUID: query.TenantUUID})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	nodes := make(map[uint64]*departmentNode, len(deps))
	for i := range deps {
		copy := deps[i]
		nodes[copy.ID] = &departmentNode{Department: copy}
	}
	var roots []*departmentNode
	for _, node := range nodes {
		if node.ParentID != nil {
			if parent, ok := nodes[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	contracts.ResponseSuccess(c, roots)
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local department service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	var req CreateDepartmentRequest
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
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		uid := uint64(tc.UserID)
		actorID = &uid
	}
	dept, err := h.service.Create(c.Request.Context(), srviam.CreateDepartmentInput{
		TenantUUID:  req.TenantUUID,
		Name:        req.Name,
		Code:        req.Code,
		ParentID:    req.ParentID,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		ActorID:     actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, dept)
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local department service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid department id")
		return
	}
	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		uid := uint64(tc.UserID)
		actorID = &uid
	}
	dept, err := h.service.Update(c.Request.Context(), id, srviam.UpdateDepartmentInput{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		ParentID:    req.ParentID,
		ActorID:     actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, dept)
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
	if h.mode == srviam.IAMAdapterModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
		return
	}
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "IAM local department service is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "local"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid department id")
		return
	}
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		uid := uint64(tc.UserID)
		actorID = &uid
	}
	if err := h.service.Delete(c.Request.Context(), id, actorID); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

func (h *DepartmentHandler) delegatedDepartments(c *gin.Context, tenantUUID string) ([]fwiamcontracts.Department, bool) {
	if h.directory == nil {
		contracts.ResponseServiceUnavailable(c, "IAM delegated directory provider is not configured", gin.H{"code": "IAM_PROVIDER_NOT_CONFIGURED", "mode": h.mode.String(), "provider": "delegated"})
		return nil, false
	}
	deps, err := h.directory.ListDepartments(c.Request.Context(), tenantUUID)
	if err != nil {
		respondIAMError(c, err)
		return nil, false
	}
	return deps, true
}

type delegatedDepartmentNode struct {
	fwiamcontracts.Department
	Children []*delegatedDepartmentNode `json:"children,omitempty"`
}

func buildDelegatedDepartmentTree(deps []fwiamcontracts.Department) []*delegatedDepartmentNode {
	nodes := make(map[string]*delegatedDepartmentNode, len(deps))
	for i := range deps {
		copy := deps[i]
		nodes[copy.ID] = &delegatedDepartmentNode{Department: copy}
	}
	roots := make([]*delegatedDepartmentNode, 0, len(deps))
	for _, node := range nodes {
		if strings.TrimSpace(node.ParentID) != "" {
			if parent, ok := nodes[node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	return roots
}
