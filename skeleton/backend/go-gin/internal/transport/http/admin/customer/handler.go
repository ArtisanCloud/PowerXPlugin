package customer

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	customerrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db   *gorm.DB
	deps *app.Deps
}

func NewHandler(deps *app.Deps) *Handler {
	if deps == nil {
		return &Handler{}
	}
	return &Handler{db: deps.DB, deps: deps}
}

type pageQuery struct {
	Page         int
	PageSize     int
	Query        string
	Status       string
	Provider     string
	CustomerUUID string
	TenantUUID   string
}

type pageResult struct {
	Items      any   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type accountDTO struct {
	ID            uint64    `json:"id"`
	CustomerUUID  string    `json:"customer_uuid"`
	TenantUUID    string    `json:"tenant_uuid,omitempty"`
	PrimaryEmail  string    `json:"primary_email,omitempty"`
	PrimaryPhone  string    `json:"primary_phone,omitempty"`
	Email         string    `json:"email,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	DisplayName   string    `json:"display_name,omitempty"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Locale        string    `json:"locale,omitempty"`
	Timezone      string    `json:"timezone,omitempty"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type createAccountRequest struct {
	TenantUUID  string         `json:"tenant_uuid"`
	Email       string         `json:"email"`
	Phone       string         `json:"phone"`
	Password    string         `json:"password"`
	DisplayName string         `json:"display_name"`
	Metadata    map[string]any `json:"metadata"`
}

func (h *Handler) Overview(c *gin.Context) {
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyOverview())
		return
	}
	tenantUUID, mismatch := admincommon.ResolveTenantUUIDStrict(c, c.Query("tenant_uuid"))
	if mismatch {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "tenant_uuid mismatch")
		return
	}
	overview := emptyOverview()
	overview["tenant_uuid"] = tenantUUID
	h.countAccounts(tenantUUID, overview)
	h.countMemberships(tenantUUID, overview)
	h.countEntries(tenantUUID, overview)
	h.countLoginEvents(tenantUUID, overview)
	contracts.ResponseSuccess(c, overview)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	query, ok := bindPageQuery(c)
	if !ok {
		return
	}
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyPage(query))
		return
	}
	query.TenantUUID, _ = admincommon.ResolveTenantUUIDStrict(c, query.TenantUUID)
	db := h.db.Model(&customermodel.CustomerAccount{})
	if query.TenantUUID != "" {
		db = db.Where("tenant_uuid = ?", query.TenantUUID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Query != "" {
		like := "%" + query.Query + "%"
		db = db.Where("customer_uuid = ? OR primary_email LIKE ? OR primary_phone LIKE ? OR email LIKE ? OR phone LIKE ? OR display_name LIKE ?", query.Query, like, like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	var accounts []customermodel.CustomerAccount
	if err := db.Order("updated_at DESC").Offset(offset(query)).Limit(query.PageSize).Find(&accounts).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	items := make([]accountDTO, 0, len(accounts))
	for _, item := range accounts {
		items = append(items, accountDTO{
			ID:            item.ID,
			CustomerUUID:  item.CustomerUUID,
			TenantUUID:    item.TenantUuid,
			PrimaryEmail:  item.PrimaryEmail,
			PrimaryPhone:  item.PrimaryPhone,
			Email:         item.Email,
			Phone:         item.Phone,
			DisplayName:   item.DisplayName,
			AvatarURL:     item.AvatarURL,
			Locale:        item.Locale,
			Timezone:      item.Timezone,
			Status:        item.Status,
			EmailVerified: item.EmailVerified,
			PhoneVerified: item.PhoneVerified,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	contracts.ResponseSuccess(c, makePage(items, query, total))
}

func (h *Handler) CreateAccount(c *gin.Context) {
	if h.db == nil {
		contracts.ResponseServiceUnavailable(c, "customer database unavailable", nil)
		return
	}
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, mismatch := admincommon.ResolveTenantUUIDStrict(c, req.TenantUUID)
	if mismatch {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "tenant_uuid mismatch")
		return
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}
	if strings.TrimSpace(req.Email) == "" && strings.TrimSpace(req.Phone) == "" {
		contracts.ResponseBadRequest(c, "email or phone is required")
		return
	}
	if len(req.Password) < 8 {
		contracts.ResponseBadRequest(c, "password must be at least 8 characters")
		return
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if displayName := strings.TrimSpace(req.DisplayName); displayName != "" {
		metadata["display_name"] = displayName
	}
	service := customersvc.NewLocalAuthService(nil, customerrepo.NewRepository(h.db))
	if h.deps != nil {
		service = customersvc.NewLocalAuthService(h.deps.Config, customerrepo.NewRepository(h.db))
	}
	out, err := service.Register(c.Request.Context(), customersvc.RegisterInput{
		TenantUUID: tenantUUID,
		Email:      strings.TrimSpace(req.Email),
		Phone:      strings.TrimSpace(req.Phone),
		Password:   req.Password,
		Metadata:   metadata,
	})
	if err != nil {
		switch {
		case strings.Contains(strings.ToLower(err.Error()), "mode"):
			contracts.ResponseBadRequest(c, "customer account creation is only available in local customer auth mode")
		case strings.Contains(strings.ToLower(err.Error()), "exists"):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "customer already exists")
		default:
			contracts.ResponseBadRequest(c, err.Error())
		}
		return
	}
	contracts.ResponseCreated(c, out)
}

func (h *Handler) ListIdentities(c *gin.Context) {
	query, ok := bindPageQuery(c)
	if !ok {
		return
	}
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyPage(query))
		return
	}
	db := h.db.Model(&customermodel.CustomerAuthIdentity{})
	if query.CustomerUUID != "" {
		db = db.Where("customer_uuid = ?", query.CustomerUUID)
	}
	if query.Provider != "" {
		db = db.Where("provider = ?", query.Provider)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Query != "" {
		like := "%" + query.Query + "%"
		db = db.Where("customer_uuid = ? OR provider_subject LIKE ? OR email LIKE ? OR phone LIKE ?", query.Query, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	var items []customermodel.CustomerAuthIdentity
	if err := db.Order("updated_at DESC").Offset(offset(query)).Limit(query.PageSize).Find(&items).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, makePage(items, query, total))
}

func (h *Handler) ListMemberships(c *gin.Context) {
	query, ok := bindPageQuery(c)
	if !ok {
		return
	}
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyPage(query))
		return
	}
	query.TenantUUID, _ = admincommon.ResolveTenantUUIDStrict(c, query.TenantUUID)
	db := h.db.Model(&customermodel.CustomerTenantMembership{})
	if query.TenantUUID != "" {
		db = db.Where("tenant_uuid = ?", query.TenantUUID)
	}
	if query.CustomerUUID != "" {
		db = db.Where("customer_uuid = ?", query.CustomerUUID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Query != "" {
		like := "%" + query.Query + "%"
		db = db.Where("membership_uuid = ? OR customer_uuid = ? OR tenant_uuid = ? OR source LIKE ?", query.Query, query.Query, query.Query, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	var items []customermodel.CustomerTenantMembership
	if err := db.Order("updated_at DESC").Offset(offset(query)).Limit(query.PageSize).Find(&items).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, makePage(items, query, total))
}

func (h *Handler) ListLoginEvents(c *gin.Context) {
	query, ok := bindPageQuery(c)
	if !ok {
		return
	}
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyPage(query))
		return
	}
	query.TenantUUID, _ = admincommon.ResolveTenantUUIDStrict(c, query.TenantUUID)
	db := h.db.Model(&customermodel.CustomerLoginEvent{})
	if query.TenantUUID != "" {
		db = db.Where("tenant_uuid = ?", query.TenantUUID)
	}
	if query.CustomerUUID != "" {
		db = db.Where("customer_uuid = ?", query.CustomerUUID)
	}
	if query.Provider != "" {
		db = db.Where("identity_provider = ?", query.Provider)
	}
	if query.Query != "" {
		like := "%" + query.Query + "%"
		db = db.Where("customer_uuid = ? OR identity_provider LIKE ? OR event_type LIKE ? OR error_code LIKE ? OR trace_id LIKE ?", query.Query, like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	var items []customermodel.CustomerLoginEvent
	if err := db.Order("created_at DESC").Offset(offset(query)).Limit(query.PageSize).Find(&items).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, makePage(items, query, total))
}

func (h *Handler) ListMiniAppEntries(c *gin.Context) {
	query, ok := bindPageQuery(c)
	if !ok {
		return
	}
	if h.db == nil {
		contracts.ResponseSuccess(c, emptyPage(query))
		return
	}
	query.TenantUUID, _ = admincommon.ResolveTenantUUIDStrict(c, query.TenantUUID)
	db := h.db.Model(&customermodel.MiniAppEntry{})
	if query.TenantUUID != "" {
		db = db.Where("tenant_uuid = ?", query.TenantUUID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Query != "" {
		like := "%" + query.Query + "%"
		db = db.Where("entry_uuid = ? OR entry_code LIKE ? OR entry_type LIKE ? OR app_key LIKE ? OR appid LIKE ? OR channel LIKE ? OR campaign LIKE ? OR brand_name LIKE ? OR org_name LIKE ?", query.Query, like, like, like, like, like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	var items []customermodel.MiniAppEntry
	if err := db.Order("updated_at DESC").Offset(offset(query)).Limit(query.PageSize).Find(&items).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, makePage(items, query, total))
}

func bindPageQuery(c *gin.Context) (pageQuery, bool) {
	query := pageQuery{
		Page:         parsePositiveInt(c.Query("page"), 1),
		PageSize:     parsePositiveInt(c.Query("page_size"), 20),
		Query:        strings.TrimSpace(c.Query("q")),
		Status:       strings.TrimSpace(c.Query("status")),
		Provider:     strings.TrimSpace(c.Query("provider")),
		CustomerUUID: strings.TrimSpace(c.Query("customer_uuid")),
		TenantUUID:   strings.TrimSpace(c.Query("tenant_uuid")),
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	_, mismatch := admincommon.ResolveTenantUUIDStrict(c, query.TenantUUID)
	if mismatch {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "tenant_uuid mismatch")
		return pageQuery{}, false
	}
	return query, true
}

func parsePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func offset(query pageQuery) int {
	return (query.Page - 1) * query.PageSize
}

func makePage(items any, query pageQuery, total int64) pageResult {
	totalPages := 0
	if query.PageSize > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(query.PageSize)))
	}
	return pageResult{
		Items:      items,
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func emptyPage(query pageQuery) pageResult {
	return makePage([]any{}, query, 0)
}

func emptyOverview() gin.H {
	return gin.H{
		"tenant_uuid":             "",
		"accounts_total":          int64(0),
		"accounts_active":         int64(0),
		"memberships_total":       int64(0),
		"memberships_active":      int64(0),
		"mini_app_entries_active": int64(0),
		"login_events_24h":        int64(0),
	}
}

func (h *Handler) countAccounts(tenantUUID string, overview gin.H) {
	db := h.db.Model(&customermodel.CustomerAccount{})
	if tenantUUID != "" {
		db = db.Where("tenant_uuid = ?", tenantUUID)
	}
	overview["accounts_total"] = count(db)
	overview["accounts_active"] = count(db.Where("status = ?", customermodel.StatusActive))
}

func (h *Handler) countMemberships(tenantUUID string, overview gin.H) {
	db := h.db.Model(&customermodel.CustomerTenantMembership{})
	if tenantUUID != "" {
		db = db.Where("tenant_uuid = ?", tenantUUID)
	}
	overview["memberships_total"] = count(db)
	overview["memberships_active"] = count(db.Where("status = ?", customermodel.StatusActive))
}

func (h *Handler) countEntries(tenantUUID string, overview gin.H) {
	db := h.db.Model(&customermodel.MiniAppEntry{})
	if tenantUUID != "" {
		db = db.Where("tenant_uuid = ?", tenantUUID)
	}
	overview["mini_app_entries_active"] = count(db.Where("status = ?", customermodel.StatusActive))
}

func (h *Handler) countLoginEvents(tenantUUID string, overview gin.H) {
	db := h.db.Model(&customermodel.CustomerLoginEvent{}).Where("created_at >= ?", time.Now().Add(-24*time.Hour))
	if tenantUUID != "" {
		db = db.Where("tenant_uuid = ?", tenantUUID)
	}
	overview["login_events_24h"] = count(db)
}

func count(db *gorm.DB) int64 {
	v := int64(0)
	if db != nil {
		_ = db.Count(&v).Error
	}
	return v
}
