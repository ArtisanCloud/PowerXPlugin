package iam

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	service *srviam.DepartmentService
	mode    srviam.IAMMode
}

type departmentNode struct {
	iamm.Department
	Children []*departmentNode `json:"children,omitempty"`
}

func NewDepartmentHandler(svc *srviam.DepartmentService, mode srviam.IAMMode) *DepartmentHandler {
	return &DepartmentHandler{service: svc, mode: mode}
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
	if h.mode == srviam.IAMModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
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
	if h.mode == srviam.IAMModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
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
	if h.mode == srviam.IAMModeDelegated {
		contracts.ResponseError(c, http.StatusMethodNotAllowed, "IAM_DELEGATED_READ_ONLY", "department write operations are not allowed in delegated mode")
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
