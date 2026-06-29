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
	Nickname      string    `json:"nickname,omitempty"`
	GivenName     string    `json:"given_name,omitempty"`
	FamilyName    string    `json:"family_name,omitempty"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Locale        string    `json:"locale,omitempty"`
	Timezone      string    `json:"timezone,omitempty"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	Metadata      any       `json:"metadata,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type createAccountRequest struct {
	TenantUUID  string         `json:"tenant_uuid"`
	Email       string         `json:"email"`
	Phone       string         `json:"phone"`
	Password    string         `json:"password"`
	DisplayName string         `json:"display_name"`
	Nickname    string         `json:"nickname"`
	GivenName   string         `json:"given_name"`
	FamilyName  string         `json:"family_name"`
	AvatarURL   string         `json:"avatar_url"`
	Locale      string         `json:"locale"`
	Timezone    string         `json:"timezone"`
	Metadata    map[string]any `json:"metadata"`
}

type updateAccountRequest struct {
	PrimaryEmail  *string        `json:"primary_email"`
	PrimaryPhone  *string        `json:"primary_phone"`
	Email         *string        `json:"email"`
	Phone         *string        `json:"phone"`
	DisplayName   *string        `json:"display_name"`
	Nickname      *string        `json:"nickname"`
	GivenName     *string        `json:"given_name"`
	FamilyName    *string        `json:"family_name"`
	AvatarURL     *string        `json:"avatar_url"`
	Locale        *string        `json:"locale"`
	Timezone      *string        `json:"timezone"`
	Status        *string        `json:"status"`
	EmailVerified *bool          `json:"email_verified"`
	PhoneVerified *bool          `json:"phone_verified"`
	Metadata      map[string]any `json:"metadata"`
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
		db = db.Where("customer_uuid = ? OR primary_email LIKE ? OR primary_phone LIKE ? OR email LIKE ? OR phone LIKE ? OR display_name LIKE ? OR nickname LIKE ? OR given_name LIKE ? OR family_name LIKE ?", query.Query, like, like, like, like, like, like, like, like)
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
		items = append(items, newAccountDTO(item))
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
	service := customersvc.NewLocalAuthService(nil, customerrepo.NewRepository(h.db))
	if h.deps != nil {
		service = customersvc.NewLocalAuthService(h.deps.Config, customerrepo.NewRepository(h.db))
	}
	out, err := service.Register(c.Request.Context(), customersvc.RegisterInput{
		TenantUUID:  tenantUUID,
		Email:       strings.TrimSpace(req.Email),
		Phone:       strings.TrimSpace(req.Phone),
		Password:    req.Password,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Nickname:    strings.TrimSpace(req.Nickname),
		GivenName:   strings.TrimSpace(req.GivenName),
		FamilyName:  strings.TrimSpace(req.FamilyName),
		AvatarURL:   strings.TrimSpace(req.AvatarURL),
		Locale:      strings.TrimSpace(req.Locale),
		Timezone:    strings.TrimSpace(req.Timezone),
		Metadata:    metadata,
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

func (h *Handler) GetAccount(c *gin.Context) {
	if h.db == nil {
		contracts.ResponseServiceUnavailable(c, "customer database unavailable", nil)
		return
	}
	account, ok := h.findAccount(c, strings.TrimSpace(c.Param("customerUUID")))
	if !ok {
		return
	}
	contracts.ResponseSuccess(c, newAccountDTO(account))
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	if h.db == nil {
		contracts.ResponseServiceUnavailable(c, "customer database unavailable", nil)
		return
	}
	account, ok := h.findAccount(c, strings.TrimSpace(c.Param("customerUUID")))
	if !ok {
		return
	}
	var req updateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	updates := map[string]any{}
	if req.PrimaryEmail != nil {
		updates["primary_email"] = strings.TrimSpace(*req.PrimaryEmail)
	}
	if req.PrimaryPhone != nil {
		updates["primary_phone"] = strings.TrimSpace(*req.PrimaryPhone)
	}
	if req.Email != nil {
		updates["email"] = strings.TrimSpace(*req.Email)
	}
	if req.Phone != nil {
		updates["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.DisplayName != nil {
		updates["display_name"] = strings.TrimSpace(*req.DisplayName)
	}
	if req.Nickname != nil {
		updates["nickname"] = strings.TrimSpace(*req.Nickname)
	}
	if req.GivenName != nil {
		updates["given_name"] = strings.TrimSpace(*req.GivenName)
	}
	if req.FamilyName != nil {
		updates["family_name"] = strings.TrimSpace(*req.FamilyName)
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*req.AvatarURL)
	}
	if req.Locale != nil {
		updates["locale"] = strings.TrimSpace(*req.Locale)
	}
	if req.Timezone != nil {
		updates["timezone"] = strings.TrimSpace(*req.Timezone)
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !isValidCustomerStatus(status) {
			contracts.ResponseBadRequest(c, "invalid status")
			return
		}
		updates["status"] = status
	}
	if req.EmailVerified != nil {
		updates["email_verified"] = *req.EmailVerified
	}
	if req.PhoneVerified != nil {
		updates["phone_verified"] = *req.PhoneVerified
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}
	if len(updates) == 0 {
		contracts.ResponseSuccess(c, newAccountDTO(account))
		return
	}
	updates["updated_at"] = time.Now()
	if err := h.db.Model(&account).Updates(updates).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	if err := h.db.Where("customer_uuid = ?", account.CustomerUUID).First(&account).Error; err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, newAccountDTO(account))
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

func (h *Handler) findAccount(c *gin.Context, customerUUID string) (customermodel.CustomerAccount, bool) {
	var account customermodel.CustomerAccount
	customerUUID = strings.ToLower(strings.TrimSpace(customerUUID))
	if customerUUID == "" {
		contracts.ResponseBadRequest(c, "customer_uuid is required")
		return account, false
	}
	db := h.db.Where("customer_uuid = ?", customerUUID)
	tenantUUID, mismatch := admincommon.ResolveTenantUUIDStrict(c, c.Query("tenant_uuid"))
	if mismatch {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "tenant_uuid mismatch")
		return account, false
	}
	if tenantUUID != "" {
		db = db.Where("tenant_uuid = ?", tenantUUID)
	}
	if err := db.First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "customer account not found")
			return account, false
		}
		contracts.ResponseInternalError(c, err)
		return account, false
	}
	return account, true
}

func newAccountDTO(item customermodel.CustomerAccount) accountDTO {
	return accountDTO{
		ID:            item.ID,
		CustomerUUID:  item.CustomerUUID,
		TenantUUID:    item.TenantUuid,
		PrimaryEmail:  item.PrimaryEmail,
		PrimaryPhone:  item.PrimaryPhone,
		Email:         item.Email,
		Phone:         item.Phone,
		DisplayName:   item.DisplayName,
		Nickname:      item.Nickname,
		GivenName:     item.GivenName,
		FamilyName:    item.FamilyName,
		AvatarURL:     item.AvatarURL,
		Locale:        item.Locale,
		Timezone:      item.Timezone,
		Status:        item.Status,
		EmailVerified: item.EmailVerified,
		PhoneVerified: item.PhoneVerified,
		Metadata:      item.Metadata,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func isValidCustomerStatus(status string) bool {
	switch status {
	case customermodel.StatusActive, customermodel.StatusPending, customermodel.StatusSuspended, customermodel.StatusDisabled:
		return true
	default:
		return false
	}
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
