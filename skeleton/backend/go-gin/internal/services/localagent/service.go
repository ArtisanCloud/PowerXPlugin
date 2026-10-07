package localagent

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	agentfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	aidto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	skilldto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	skillfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/localagent"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	builtinskills "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/skills"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type actorKey struct{}
type agentKey struct{}

// WithActor carries a validated authentication claim, never a request payload.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

type AI interface {
	ListLLMModels(context.Context, string) (*aidto.ListLLMModelsOutput, error)
	LLMInvoke(context.Context, aidto.LLMInvokeInput) (*aidto.LLMInvokeOutput, error)
}
type Service struct {
	db     *gorm.DB
	ai     AI
	skills skillfw.Invoker
}

func New(db *gorm.DB, ai AI, skills skillfw.Invoker) *Service {
	return &Service{db: db, ai: ai, skills: skills}
}

var _ agentfw.ManagementService = (*Service)(nil)
var _ skillfw.ManagementService = (*Service)(nil)

type Error struct {
	Code   string
	Status int
}

func (e *Error) Error() string              { return e.Code }
func failure(code string, status int) error { return &Error{Code: code, Status: status} }
func scope(ctx context.Context) (string, error) {
	t, ok := authx.TenantUUIDFromContext(ctx)
	if !ok || !validUUID(t) {
		return "", failure("LOCAL_AGENT_UNAUTHORIZED", 401)
	}
	return t, nil
}
func validUUID(id string) bool {
	v, e := uuid.Parse(id)
	return e == nil && v != uuid.Nil && v.String() == id
}

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func validate(key, name, status string) error {
	if !keyPattern.MatchString(key) || strings.TrimSpace(name) == "" || len(name) > 255 || (status != "active" && status != "inactive") {
		return failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
	}
	return nil
}
func normalize(q skillfw.DefinitionQuery) skillfw.DefinitionQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}
func filtered(db *gorm.DB, t string, q skillfw.DefinitionQuery) *gorm.DB {
	db = db.Where("tenant_uuid = ?", t)
	if q.Query != "" {
		db = db.Where("(LOWER(name) LIKE ? OR LOWER(key) LIKE ?)", "%"+strings.ToLower(q.Query)+"%", "%"+strings.ToLower(q.Query)+"%")
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	return db
}
func skillView(r model.Skill) skillfw.Definition {
	return skillfw.Definition{UUID: r.UUID, Key: r.Key, Name: r.Name, Description: r.Description, Version: r.Version, Status: r.Status, Executor: r.Executor, ModelKey: r.ModelKey, Prompt: r.Prompt, UpdatedAt: r.UpdatedAt}
}
func agentView(r model.Agent) agentfw.Definition {
	ids := append([]string{}, r.SkillUUIDs...)
	return agentfw.Definition{UUID: r.UUID, Key: r.Key, Name: r.Name, Description: r.Description, Status: r.Status, ModelKey: r.ModelKey, Persona: r.Persona, Prompt: r.Prompt, SkillUUIDs: ids, UpdatedAt: r.UpdatedAt}
}
func (s *Service) ListSkills(ctx context.Context, q skillfw.DefinitionQuery) (*skillfw.DefinitionPage, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	q = normalize(q)
	out := &skillfw.DefinitionPage{Items: []skillfw.Definition{}, Page: q.Page, PageSize: q.PageSize}
	db := filtered(s.db.WithContext(ctx).Model(&model.Skill{}), t, q)
	if e = db.Count(&out.Total).Error; e != nil {
		return nil, e
	}
	var rows []model.Skill
	if e = db.Order("updated_at DESC, uuid ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; e != nil {
		return nil, e
	}
	for _, r := range rows {
		out.Items = append(out.Items, skillView(r))
	}
	return out, nil
}
func (s *Service) ListAgents(ctx context.Context, q skillfw.DefinitionQuery) (*agentfw.DefinitionPage, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	q = normalize(q)
	out := &agentfw.DefinitionPage{Items: []agentfw.Definition{}, Page: q.Page, PageSize: q.PageSize}
	db := filtered(s.db.WithContext(ctx).Model(&model.Agent{}), t, q)
	if e = db.Count(&out.Total).Error; e != nil {
		return nil, e
	}
	var rows []model.Agent
	if e = db.Order("updated_at DESC, uuid ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error; e != nil {
		return nil, e
	}
	for _, r := range rows {
		out.Items = append(out.Items, agentView(r))
	}
	return out, nil
}
func (s *Service) Models(ctx context.Context) (*aidto.ListLLMModelsOutput, error) {
	if _, e := scope(ctx); e != nil {
		return nil, e
	}
	if s.ai == nil {
		return nil, failure("LOCAL_MODEL_UNAVAILABLE", 503)
	}
	return s.ai.ListLLMModels(ctx, "")
}
func (s *Service) modelAvailable(ctx context.Context, key string) error {
	models, e := s.Models(ctx)
	if e != nil {
		return e
	}
	if models != nil {
		for _, m := range models.Items {
			if m.ModelKey == key && m.Configured {
				return nil
			}
		}
	}
	return failure("LOCAL_MODEL_UNAVAILABLE", 422)
}
func (s *Service) SaveSkill(ctx context.Context, id string, in skillfw.DefinitionInput) (*skillfw.Definition, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	if e = validate(in.Key, in.Name, in.Status); e != nil {
		return nil, e
	}
	if !keyPattern.MatchString(in.Version) || (in.Executor != "template" && in.Executor != "prompt") || len(in.Prompt) > 32000 {
		return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
	}
	if in.Executor == "prompt" && in.Status == "active" {
		if strings.TrimSpace(in.Prompt) == "" {
			return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
		}
		if e = s.modelAvailable(ctx, in.ModelKey); e != nil {
			return nil, e
		}
	}
	var r model.Skill
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id != "" {
			if !validUUID(id) {
				return failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
			}
			if err := tx.Where("uuid = ? AND tenant_uuid = ?", id, t).First(&r).Error; err != nil {
				return err
			}
			if r.Key != in.Key || r.Version != in.Version {
				return failure("LOCAL_AGENT_IMMUTABLE_KEY", 400)
			}
		} else {
			r.UUID = uuid.NewString()
			r.TenantUUID = t
		}
		var n int64
		if err := tx.Model(&model.Skill{}).Where("tenant_uuid = ? AND key = ? AND version = ? AND uuid <> ?", t, in.Key, in.Version, r.UUID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return failure("LOCAL_AGENT_CONFLICT", 409)
		}
		r.Key = in.Key
		r.Name = strings.TrimSpace(in.Name)
		r.Version = in.Version
		r.Description = in.Description
		r.Status = in.Status
		r.Executor = in.Executor
		r.ModelKey = in.ModelKey
		r.Prompt = in.Prompt
		return tx.Save(&r).Error
	})
	if e != nil {
		return nil, e
	}
	out := skillView(r)
	return &out, nil
}
func (s *Service) SaveAgent(ctx context.Context, id string, in agentfw.DefinitionInput) (*agentfw.Definition, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	if e = validate(in.Key, in.Name, in.Status); e != nil {
		return nil, e
	}
	if len(in.SkillUUIDs) > 64 || len(in.Prompt) > 32000 || len(in.Persona) > 8000 {
		return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
	}
	if in.Status == "active" {
		if e = s.modelAvailable(ctx, in.ModelKey); e != nil {
			return nil, e
		}
	}
	var r model.Agent
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id != "" {
			if !validUUID(id) {
				return failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
			}
			if err := tx.Where("uuid = ? AND tenant_uuid = ?", id, t).First(&r).Error; err != nil {
				return err
			}
			if r.Key != in.Key {
				return failure("LOCAL_AGENT_IMMUTABLE_KEY", 400)
			}
		} else {
			r.UUID = uuid.NewString()
			r.TenantUUID = t
		}
		seen := map[string]bool{}
		for _, sid := range in.SkillUUIDs {
			if !validUUID(sid) || seen[sid] {
				return failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
			}
			seen[sid] = true
			var skill model.Skill
			if err := tx.Where("uuid = ? AND tenant_uuid = ?", sid, t).First(&skill).Error; err != nil {
				return err
			}
			if in.Status == "active" && skill.Status != "active" {
				return failure("LOCAL_SKILL_DISABLED", 409)
			}
		}
		var n int64
		if err := tx.Model(&model.Agent{}).Where("tenant_uuid = ? AND key = ? AND uuid <> ?", t, in.Key, r.UUID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return failure("LOCAL_AGENT_CONFLICT", 409)
		}
		r.Key = in.Key
		r.Name = strings.TrimSpace(in.Name)
		r.Description = in.Description
		r.Status = in.Status
		r.ModelKey = in.ModelKey
		r.Persona = in.Persona
		r.Prompt = in.Prompt
		r.SkillUUIDs = append([]string{}, in.SkillUUIDs...)
		return tx.Save(&r).Error
	})
	if e != nil {
		return nil, e
	}
	out := agentView(r)
	return &out, nil
}
func (s *Service) InvokeSkill(ctx context.Context, id string, input map[string]any) (map[string]any, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	if !validUUID(id) {
		return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
	}
	var r model.Skill
	if e = s.db.WithContext(ctx).Where("uuid = ? AND tenant_uuid = ?", id, t).First(&r).Error; e != nil {
		return nil, e
	}
	if r.Status != "active" {
		return nil, failure("LOCAL_SKILL_DISABLED", 409)
	}
	trace := uuid.NewString()
	switch r.Executor {
	case "template":
		actor, _ := ctx.Value(actorKey{}).(string)
		if !validUUID(actor) {
			return nil, failure("LOCAL_AGENT_UNAUTHORIZED", 401)
		}
		agentID, _ := ctx.Value(agentKey{}).(string)
		if agentID == "" {
			agentID = "local-admin-debug"
		}
		if s.skills == nil {
			return nil, failure("LOCAL_EXECUTOR_UNAVAILABLE", 503)
		}
		out, err := s.skills.Invoke(ctx, skilldto.InvokeInput{SkillID: builtinskills.TemplateSkillID, Version: "1.0.0", Payload: input, Context: map[string]any{"trace_id": trace, "request_id": trace, "channel": "admin", "locale": "zh-CN", "user_uuid": actor, "agent_id": agentID, "session_id": trace}})
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, failure("LOCAL_EXECUTOR_UNAVAILABLE", 503)
		}
		return map[string]any{"trace_id": trace, "status": out.Status, "result": out.Result}, nil
	case "prompt":
		if e = s.modelAvailable(ctx, r.ModelKey); e != nil {
			return nil, e
		}
		raw, err := json.Marshal(input)
		if err != nil {
			return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
		}
		out, err := s.ai.LLMInvoke(ctx, aidto.LLMInvokeInput{ModelKey: r.ModelKey, Inputs: []aidto.ContentItem{
			{Role: "system", Type: "text", Content: r.Prompt},
			{Role: "user", Type: "text", Content: string(raw)},
		}})
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, failure("LOCAL_MODEL_UNAVAILABLE", 503)
		}
		return map[string]any{"trace_id": trace, "status": "completed", "result": map[string]any{"text": out.Text}}, nil
	default:
		return nil, failure("LOCAL_EXECUTOR_UNAVAILABLE", 503)
	}
}
func (s *Service) DebugAgent(ctx context.Context, id string, in agentfw.DebugInput) (*agentfw.DebugOutput, error) {
	t, e := scope(ctx)
	if e != nil {
		return nil, e
	}
	if !validUUID(id) || strings.TrimSpace(in.Message) == "" || len(in.Message) > 32000 {
		return nil, failure("LOCAL_AGENT_INVALID_ARGUMENT", 400)
	}
	var r model.Agent
	if e = s.db.WithContext(ctx).Where("uuid = ? AND tenant_uuid = ?", id, t).First(&r).Error; e != nil {
		return nil, e
	}
	if r.Status != "active" {
		return nil, failure("LOCAL_AGENT_DISABLED", 409)
	}
	if e = s.modelAvailable(ctx, r.ModelKey); e != nil {
		return nil, e
	}
	out := &agentfw.DebugOutput{TraceID: uuid.NewString()}
	inputs := []aidto.ContentItem{
		{Role: "system", Type: "text", Content: r.Persona + "\n" + r.Prompt},
		{Role: "user", Type: "text", Content: in.Message},
	}
	if in.SkillUUID != "" {
		allowed := false
		for _, sid := range r.SkillUUIDs {
			if sid == in.SkillUUID {
				allowed = true
			}
		}
		if !allowed {
			return nil, failure("LOCAL_SKILL_NOT_BOUND", 403)
		}
		out.SkillResult, e = s.InvokeSkill(context.WithValue(ctx, agentKey{}, r.UUID), in.SkillUUID, in.SkillInput)
		if e != nil {
			return nil, e
		}
		raw, _ := json.Marshal(out.SkillResult)
		inputs = append(inputs, aidto.ContentItem{Role: "user", Type: "text", Content: string(raw)})
	}
	answer, e := s.ai.LLMInvoke(ctx, aidto.LLMInvokeInput{ModelKey: r.ModelKey, Inputs: inputs})
	if e != nil {
		return nil, e
	}
	if answer == nil {
		return nil, failure("LOCAL_MODEL_UNAVAILABLE", 503)
	}
	out.Text = answer.Text
	return out, nil
}
func ErrorStatus(err error) (int, string) {
	var e *Error
	if errors.As(err, &e) {
		return e.Status, e.Code
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 404, "LOCAL_AGENT_NOT_FOUND"
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return 409, "LOCAL_AGENT_CONFLICT"
	}
	return 500, "LOCAL_AGENT_OPERATION_FAILED"
}
