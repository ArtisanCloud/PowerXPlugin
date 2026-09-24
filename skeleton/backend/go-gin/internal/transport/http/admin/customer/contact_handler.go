package customer

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	contactfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type ContactHandler struct{ deps *app.Deps }

func NewContactHandler(deps *app.Deps) *ContactHandler { return &ContactHandler{deps: deps} }

type contactCreateRequest struct {
	DisplayName    string           `json:"display_name" binding:"required,max=128"`
	GivenName      string           `json:"given_name" binding:"max=128"`
	FamilyName     string           `json:"family_name" binding:"max=128"`
	Status         contactfw.Status `json:"status" binding:"required,oneof=active inactive temporary"`
	Roles          []contactfw.Role `json:"roles"`
	Tags           []string         `json:"tags"`
	CreationIntent string           `json:"creation_intent" binding:"required,oneof=explicit_create explicit_temporary"`
}
type contactUpdateRequest struct {
	DisplayName *string           `json:"display_name" binding:"omitempty,max=128"`
	GivenName   *string           `json:"given_name" binding:"omitempty,max=128"`
	FamilyName  *string           `json:"family_name" binding:"omitempty,max=128"`
	Status      *contactfw.Status `json:"status" binding:"omitempty,oneof=active inactive temporary"`
	Roles       *[]contactfw.Role `json:"roles"`
	Tags        *[]string         `json:"tags"`
}
type contactIdentityRequest struct {
	ChannelDictionaryItemUUID string `json:"channel_dictionary_item_uuid" binding:"required,uuid4"`
	ExternalSubject           string `json:"external_subject" binding:"required,max=255"`
}
type migrateContactIdentityChannelRequest struct {
	ChannelDictionaryItemUUID string `json:"channel_dictionary_item_uuid" binding:"required,uuid4"`
}

func (h *ContactHandler) List(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	page, err := store.ListByCustomer(ctx, contactfw.ListByCustomerInput{CustomerUUID: customerUUID, Query: strings.TrimSpace(c.Query("q")), Status: contactStatus(c.Query("status")), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", contactfw.DefaultPageSize)})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseSuccess(c, pageResult{Items: page.Items, Page: page.Page, PageSize: page.PageSize, Total: page.Total, TotalPages: int((page.Total + int64(page.PageSize) - 1) / int64(page.PageSize))})
}

func (h *ContactHandler) Create(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	var req contactCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return
	}
	item, err := store.Create(ctx, contactfw.CreateContactInput{CustomerUUID: customerUUID, DisplayName: req.DisplayName, GivenName: req.GivenName, FamilyName: req.FamilyName, Status: req.Status, Roles: req.Roles, Tags: req.Tags, CreationIntent: contactfw.CreationIntent(req.CreationIntent)})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *ContactHandler) Get(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	item, err := store.Get(ctx, contactfw.GetContactInput{CustomerUUID: customerUUID, ContactUUID: strings.TrimSpace(c.Param("contactUUID"))})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ContactHandler) Update(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	var req contactUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return
	}
	item, err := store.Update(ctx, contactfw.UpdateContactInput{CustomerUUID: customerUUID, ContactUUID: strings.TrimSpace(c.Param("contactUUID")), DisplayName: req.DisplayName, GivenName: req.GivenName, FamilyName: req.FamilyName, Status: req.Status, Roles: req.Roles, Tags: req.Tags})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ContactHandler) ResolveIdentity(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	var req contactIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return
	}
	item, err := store.ResolveIdentity(ctx, contactfw.ResolveContactIdentityInput{CustomerUUID: customerUUID, ChannelDictionaryItemUUID: req.ChannelDictionaryItemUUID, ExternalSubject: req.ExternalSubject})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ContactHandler) BindIdentity(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	var req contactIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return
	}
	item, err := store.BindIdentity(ctx, contactfw.BindContactIdentityInput{CustomerUUID: customerUUID, ContactUUID: strings.TrimSpace(c.Param("contactUUID")), ChannelDictionaryItemUUID: req.ChannelDictionaryItemUUID, ExternalSubject: req.ExternalSubject})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *ContactHandler) MigrateIdentityChannel(c *gin.Context) {
	store, ctx, customerUUID, ok := h.storeContext(c)
	if !ok {
		return
	}
	migrator, ok := store.(contactfw.IdentityChannelMigrator)
	if !ok {
		contracts.ResponseError(c, http.StatusServiceUnavailable, string(contactfw.CodeLocalUnavailable), string(contactfw.CodeLocalUnavailable))
		return
	}
	var req migrateContactIdentityChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return
	}
	item, err := migrator.MigrateIdentityChannel(ctx, contactfw.MigrateContactIdentityChannelInput{CustomerUUID: customerUUID, ContactUUID: strings.TrimSpace(c.Param("contactUUID")), IdentityUUID: strings.TrimSpace(c.Param("identityUUID")), ChannelDictionaryItemUUID: req.ChannelDictionaryItemUUID})
	if err != nil {
		h.respondError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ContactHandler) storeContext(c *gin.Context) (contactfw.Store, context.Context, string, bool) {
	if h == nil || h.deps == nil {
		contracts.ResponseError(c, http.StatusServiceUnavailable, string(contactfw.CodeRuntimeRequired), string(contactfw.CodeRuntimeRequired))
		return nil, nil, "", false
	}
	runtime := h.deps.ContactRuntime
	switch strings.TrimSpace(c.Query("framework_debug_route")) {
	case "", "runtime":
	case "local":
		runtime = h.deps.ContactDebugLocalRuntime
	case "delegated":
		runtime = h.deps.ContactDebugDelegatedRuntime
	default:
		contracts.ResponseError(c, http.StatusBadRequest, string(contactfw.CodeInvalidArgument), string(contactfw.CodeInvalidArgument))
		return nil, nil, "", false
	}
	if runtime == nil {
		contracts.ResponseError(c, http.StatusServiceUnavailable, string(contactfw.CodeDelegateUnavailable), string(contactfw.CodeDelegateUnavailable))
		return nil, nil, "", false
	}
	store, err := runtime.Store()
	if err != nil {
		h.respondError(c, err)
		return nil, nil, "", false
	}
	tenantUUID, mismatch := admincommon.ResolveTenantUUIDStrict(c, c.Query("tenant_uuid"))
	if mismatch || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseError(c, http.StatusForbidden, string(contactfw.CodeCustomerMismatch), string(contactfw.CodeCustomerMismatch))
		return nil, nil, "", false
	}
	return store, customerfw.WithTenantUUID(c.Request.Context(), tenantUUID), strings.TrimSpace(c.Param("customerUUID")), true
}

func (h *ContactHandler) respondError(c *gin.Context, err error) {
	code := contactfw.CodeOf(err)
	if code == "" {
		code = contactfw.CodeLocalUnavailable
	}
	status := http.StatusBadRequest
	switch code {
	case contactfw.CodeNotFound, contactfw.CodeIdentityNotFound:
		status = http.StatusNotFound
	case contactfw.CodeCustomerMismatch, contactfw.CodeCapabilityForbidden:
		status = http.StatusForbidden
	case contactfw.CodeIdentityConflict:
		status = http.StatusConflict
	case contactfw.CodeDelegateUnavailable, contactfw.CodeLocalUnavailable, contactfw.CodeRuntimeRequired:
		status = http.StatusServiceUnavailable
	case contactfw.CodeChannelDictionaryInvalid, contactfw.CodeIdentityChannelMigrationRequired:
		status = http.StatusConflict
	}
	contracts.ResponseError(c, status, string(code), string(code))
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func contactStatus(raw string) *contactfw.Status {
	value := contactfw.Status(strings.TrimSpace(raw))
	if value == "" || value == "all" {
		return nil
	}
	return &value
}
