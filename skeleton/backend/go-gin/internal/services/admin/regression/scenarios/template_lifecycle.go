package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	Templates "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/templates"
	"gorm.io/gorm"
	"net/http"
	"time"
)

const SuiteUUID = "11111111-1111-4111-8111-111111111111"
const TemplateCaseUUID = "22222222-2222-4222-8222-222222222222"
const TemplateCreateStep = "33333333-3333-4333-8333-333333333331"
const TemplateUpdateStep = "33333333-3333-4333-8333-333333333332"
const TemplateReadStep = "33333333-3333-4333-8333-333333333333"

type Dependencies struct {
	Templates    *Templates.TemplateService
	TemplateHTTP http.Handler
}
type TemplateLifecycle struct{}

func (*TemplateLifecycle) Definition() Regression.CaseDefinition {
	return Regression.CaseDefinition{UUID: TemplateCaseUUID, SuiteUUID: SuiteUUID, Key: "template.lifecycle", Version: "v1", TitleKey: "regression.scenarios.template.title", DescriptionKey: "regression.scenarios.template.description", Tags: []string{"template", "lifecycle"}, SupportedProfileKinds: []string{"controlled"}, InputSchema: Regression.JSON(map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string", "minLength": 1}, "description": map[string]any{"type": "string", "minLength": 1}, "content": map[string]any{"type": "string", "minLength": 1}, "updated_content": map[string]any{"type": "string", "minLength": 1}}, "required": []string{"name", "description", "content", "updated_content"}, "additionalProperties": false}), ExecutionTimeoutMS: 30000, CleanupTimeoutMS: 10000, RequiredAssertionKeys: []string{"template.created", "template.updated", "template.readback"}, Steps: []Regression.StepDefinition{{UUID: TemplateCreateStep, Key: "create", TitleKey: "regression.steps.create", Phase: "execute", Boundary: "service"}, {UUID: TemplateUpdateStep, Key: "update", TitleKey: "regression.steps.update", Phase: "execute", Boundary: "service"}, {UUID: TemplateReadStep, Key: "readback", TitleKey: "regression.steps.readback", Phase: "verify", Boundary: "service"}}}
}
func dep(s *Regression.Scope) (*Dependencies, error) {
	d, ok := s.Dependencies.(*Dependencies)
	if !ok || d == nil || d.Templates == nil {
		return nil, Regression.NewError(Regression.DependencyBlocked)
	}
	return d, nil
}
func (*TemplateLifecycle) Preflight(ctx context.Context, s *Regression.Scope) error {
	_, err := dep(s)
	return err
}
func (*TemplateLifecycle) Prepare(context.Context, *Regression.Scope) error { return nil }
func (*TemplateLifecycle) Execute(ctx context.Context, s *Regression.Scope) error {
	d, err := dep(s)
	if err != nil {
		return err
	}
	var input struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Content        string `json:"content"`
		UpdatedContent string `json:"updated_content"`
	}
	if err = json.Unmarshal(s.Input, &input); err != nil {
		return Regression.NewError(Regression.InvalidRequest)
	}
	ctx = Middleware.ContextWithTenantUUID(ctx, s.Actor.TenantUUID)
	var resource string
	if err = s.Recorder.Step(ctx, TemplateCreateStep, func(ctx context.Context) error {
		v, e := d.Templates.Create(ctx, input.Name, input.Description, input.Content)
		if e != nil {
			return e
		}
		resource = v.UUID
		if e = s.Recorder.Register(ctx, Regression.FixtureResource{ResourceType: "template", ResourceUUID: resource, DisplayName: v.Name, RoleKey: "regression.resourceRoles.fixture", CleanupKey: "template.delete", CleanupParameters: Regression.JSON(map[string]any{})}); e != nil {
			_ = d.Templates.Delete(context.WithoutCancel(ctx), resource)
			return e
		}
		return s.Recorder.Assert(ctx, "template.created", "regression.assertions.created", "passed", input.Content, v.Content, proof("template.create", resource, v.Content))
	}); err != nil {
		return err
	}
	if err = s.Recorder.Step(ctx, TemplateUpdateStep, func(ctx context.Context) error {
		v, e := d.Templates.Update(ctx, resource, input.Name, input.Description, input.UpdatedContent)
		if e != nil {
			return e
		}
		return s.Recorder.Assert(ctx, "template.updated", "regression.assertions.updated", "passed", input.UpdatedContent, v.Content, proof("template.update", resource, v.Content))
	}); err != nil {
		return err
	}
	return s.Recorder.Step(ctx, TemplateReadStep, func(ctx context.Context) error {
		v, e := d.Templates.GetByUUID(ctx, resource)
		if e != nil {
			return e
		}
		return s.Recorder.Assert(ctx, "template.readback", "regression.assertions.readback", "passed", input.UpdatedContent, v.Content, proof("template.read", resource, v.Content))
	})
}
func (*TemplateLifecycle) Cleanup(ctx context.Context, s *Regression.Scope) error {
	d, err := dep(s)
	if err != nil {
		return err
	}
	ctx = Middleware.ContextWithTenantUUID(ctx, s.Actor.TenantUUID)
	for _, f := range s.Recorder.ListOwned() {
		if f.CleanupState == "succeeded" {
			continue
		}
		if f.ResourceType != "template" || f.CleanupKey != "template.delete" || f.TenantUUID != s.Actor.TenantUUID || f.RunUUID != s.RunUUID {
			return Regression.NewError(Regression.CleanupUnavailable)
		}
		err = d.Templates.Delete(ctx, f.ResourceUUID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if _, readErr := d.Templates.GetByUUID(ctx, f.ResourceUUID); !errors.Is(readErr, gorm.ErrRecordNotFound) {
			return Regression.NewError(Regression.CleanupFailed)
		}
		if err = s.Recorder.Cleaned(ctx, f.ResourceUUID, proof("template.read_after_delete", f.ResourceUUID, map[string]any{"exists": false})); err != nil {
			return err
		}
	}
	return nil
}
func proof(source, id string, data any) []Regression.Evidence {
	return []Regression.Evidence{{Kind: "readback", Boundary: "service", SourceRef: source, ResourceUUID: id, ObservedAt: time.Now().UTC(), SanitizedData: Regression.JSON(data)}}
}
