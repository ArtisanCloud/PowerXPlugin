package skills

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	runtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/templates"
	"gorm.io/gorm"
)

type TemplateSkillExecutor struct {
	templates *srvtemplates.TemplateService
}

func NewTemplateSkillExecutor(db *gorm.DB) (*TemplateSkillExecutor, error) {
	if db == nil {
		return nil, runtime.NewError(runtime.ErrCodeExecutorUnavailable, "template skill executor requires database")
	}
	return &TemplateSkillExecutor{templates: srvtemplates.NewTemplateService(db)}, nil
}

func (e *TemplateSkillExecutor) Execute(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	select {
	case <-ctx.Done():
		return runtime.PluginSkillResult{}, ctx.Err()
	default:
	}
	if e == nil || e.templates == nil {
		return runtime.PluginSkillResult{}, runtime.NewError(runtime.ErrCodeExecutorUnavailable, "template service is not configured")
	}
	ctx = authx.ContextWithTenantUUID(ctx, inv.Context.TenantUUID)
	action := strings.ToLower(strings.TrimSpace(stringFromMap(inv.Input, "action")))
	if action == "" {
		return runtime.PluginSkillResult{}, invalidInvocation("action is required", "action")
	}
	switch action {
	case "create":
		return e.create(ctx, inv)
	case "get":
		return e.get(ctx, inv)
	case "update":
		return e.update(ctx, inv)
	case "delete":
		return e.delete(ctx, inv)
	case "list":
		return e.list(ctx, inv)
	default:
		return runtime.PluginSkillResult{}, invalidInvocation("unsupported action: "+action, "action")
	}
}

func (e *TemplateSkillExecutor) create(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	tpl, err := templatePayload(inv.Input)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	created, err := e.templates.Create(ctx, tpl.Title, tpl.Description, tpl.Content)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	return runtime.SuccessResult(inv, runtime.ResultCompleted, "模板已创建", map[string]any{
		"action":      "create",
		"template_id": fmt.Sprintf("%d", created.ID),
		"template":    templateResult(created.ID, created.Name, created.Description, created.Content),
	}), nil
}

func (e *TemplateSkillExecutor) get(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	id, err := templateID(inv.Input)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	tpl, err := e.templates.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return runtime.PluginSkillResult{}, runtime.NewError(runtime.ErrCodeNotFound, "template not found")
		}
		return runtime.PluginSkillResult{}, err
	}
	return runtime.SuccessResult(inv, runtime.ResultCompleted, "模板查询完成", map[string]any{
		"action":      "get",
		"template_id": fmt.Sprintf("%d", tpl.ID),
		"template":    templateResult(tpl.ID, tpl.Name, tpl.Description, tpl.Content),
	}), nil
}

func (e *TemplateSkillExecutor) update(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	id, err := templateID(inv.Input)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	tpl, err := templatePayload(inv.Input)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	updated, err := e.templates.Update(ctx, id, tpl.Title, tpl.Description, tpl.Content)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	return runtime.SuccessResult(inv, runtime.ResultCompleted, "模板已更新", map[string]any{
		"action":      "update",
		"template_id": fmt.Sprintf("%d", updated.ID),
		"template":    templateResult(updated.ID, updated.Name, updated.Description, updated.Content),
	}), nil
}

func (e *TemplateSkillExecutor) delete(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	id, err := templateID(inv.Input)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	if err := e.templates.Delete(ctx, id); err != nil {
		return runtime.PluginSkillResult{}, err
	}
	return runtime.SuccessResult(inv, runtime.ResultCompleted, "模板已删除", map[string]any{
		"action":      "delete",
		"template_id": fmt.Sprintf("%d", id),
	}), nil
}

func (e *TemplateSkillExecutor) list(ctx context.Context, inv runtime.PluginSkillInvocation) (runtime.PluginSkillResult, error) {
	q := stringFromMap(inv.Input, "q")
	page := intFromMap(inv.Input, "page", 1)
	pageSize := intFromMap(inv.Input, "page_size", 20)
	res, err := e.templates.List(ctx, q, page, pageSize)
	if err != nil {
		return runtime.PluginSkillResult{}, err
	}
	items := make([]map[string]any, 0, len(res.List))
	for _, tpl := range res.List {
		items = append(items, templateResult(tpl.ID, tpl.Name, tpl.Description, tpl.Content))
	}
	return runtime.SuccessResult(inv, runtime.ResultCompleted, "模板列表查询完成", map[string]any{
		"action":    "list",
		"items":     items,
		"page":      res.PageIndex,
		"page_size": res.PageSize,
		"total":     res.Total,
	}), nil
}

type templateInput struct {
	Title       string
	Description string
	Content     string
}

func templatePayload(input map[string]any) (templateInput, error) {
	raw, ok := input["template"].(map[string]any)
	if !ok || raw == nil {
		return templateInput{}, invalidInvocation("template is required", "template")
	}
	out := templateInput{
		Title:       strings.TrimSpace(stringFromMap(raw, "title")),
		Description: strings.TrimSpace(stringFromMap(raw, "description")),
		Content:     strings.TrimSpace(stringFromMap(raw, "content")),
	}
	missing := make([]string, 0, 3)
	if out.Title == "" {
		missing = append(missing, "template.title")
	}
	if out.Description == "" {
		missing = append(missing, "template.description")
	}
	if out.Content == "" {
		missing = append(missing, "template.content")
	}
	if len(missing) > 0 {
		err := runtime.NewError(runtime.ErrCodeInvalidInvocation, "template fields are required")
		err.Details = map[string]any{"missing_fields": missing}
		return templateInput{}, err
	}
	return out, nil
}

func templateID(input map[string]any) (uint64, error) {
	raw := stringFromMap(input, "template_id")
	if raw == "" {
		raw = stringFromMap(input, "id")
	}
	if raw == "" {
		return 0, invalidInvocation("template_id is required", "template_id")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, invalidInvocation("template_id must be a positive integer", "template_id")
	}
	return id, nil
}

func invalidInvocation(message, field string) *runtime.SkillError {
	err := runtime.NewError(runtime.ErrCodeInvalidInvocation, message)
	err.Details = map[string]any{"field": field}
	return err
}

func stringFromMap(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	switch v := input[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case float64:
		if v == float64(uint64(v)) {
			return strconv.FormatUint(uint64(v), 10)
		}
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	}
	return ""
}

func intFromMap(input map[string]any, key string, fallback int) int {
	raw := stringFromMap(input, key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func templateResult(id uint64, name, description, content string) map[string]any {
	return map[string]any{
		"id":          id,
		"title":       name,
		"description": description,
		"content":     content,
	}
}
