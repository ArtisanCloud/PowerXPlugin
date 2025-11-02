package templates

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	"github.com/powerx-plugin/framework/backend/go/router"
)

// Handler exposes HTTP endpoints for template operations.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	if svc == nil {
		svc = NewService(nil)
	}
	return &Handler{svc: svc}
}

func (h *Handler) List() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		page, _ := strconv.Atoi(ctx.Query("page"))
		pageSize, _ := strconv.Atoi(ctx.Query("page_size"))
		query := ctx.Query("q")

		res, err := h.svc.List(ctx.Context(), query, page, pageSize)
		if err != nil {
			h.handleError(ctx, err)
			return
		}
		router.RespondSuccess(ctx, http.StatusOK, res, "")
	}
}

func (h *Handler) Get() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		id, err := parseID(ctx.Param("id"))
		if err != nil {
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid id", nil)
			return
		}
		tpl, err := h.svc.Get(ctx.Context(), id)
		if err != nil {
			h.handleError(ctx, err)
			return
		}
		router.RespondSuccess(ctx, http.StatusOK, tpl, "")
	}
}

func (h *Handler) Create() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		var payload struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
		}
		if err := ctx.BindJSON(&payload); err != nil {
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid body", nil)
			return
		}
		tpl, err := h.svc.Create(ctx.Context(), payload.Name, payload.Description, payload.Content)
		if err != nil {
			h.handleError(ctx, err)
			return
		}
		router.RespondSuccess(ctx, http.StatusCreated, tpl, "")
	}
}

func (h *Handler) Update() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		id, err := parseID(ctx.Param("id"))
		if err != nil {
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid id", nil)
			return
		}
		var payload struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
		}
		if err := ctx.BindJSON(&payload); err != nil {
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid body", nil)
			return
		}
		tpl, err := h.svc.Update(ctx.Context(), id, payload.Name, payload.Description, payload.Content)
		if err != nil {
			h.handleError(ctx, err)
			return
		}
		router.RespondSuccess(ctx, http.StatusOK, tpl, "")
	}
}

func (h *Handler) Delete() bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		id, err := parseID(ctx.Param("id"))
		if err != nil {
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid id", nil)
			return
		}
		if err := h.svc.Delete(ctx.Context(), id); err != nil {
			handleErr := translateError(err)
			router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", handleErr.Error(), nil)
			return
		}
		router.RespondSuccess(ctx, http.StatusOK, map[string]bool{"ok": true}, "")
	}
}

func (h *Handler) handleError(ctx bootstrap.Context, err error) {
	switch {
	case errors.Is(err, errTenantRequired):
		router.RespondError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
	case errors.Is(err, errInvalidInput):
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, errNotFound):
		router.RespondError(ctx, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	default:
		router.RespondError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}

func parseID(raw string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
}
