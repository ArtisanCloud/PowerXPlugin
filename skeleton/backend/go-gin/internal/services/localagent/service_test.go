package localagent

import (
	"context"
	"path/filepath"
	"testing"

	agentfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	aidto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	skilldto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	skillfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/localagent"
	templatemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	builtinskills "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/skills"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeAI struct {
	calls int
	last  aidto.LLMInvokeInput
}

func (f *fakeAI) ListLLMModels(context.Context, string) (*aidto.ListLLMModelsOutput, error) {
	return &aidto.ListLLMModelsOutput{Items: []aidto.LLMModel{{ModelKey: "test.model", Configured: true}}}, nil
}
func (f *fakeAI) LLMInvoke(_ context.Context, in aidto.LLMInvokeInput) (*aidto.LLMInvokeOutput, error) {
	f.calls++
	f.last = in
	return &aidto.LLMInvokeOutput{Text: "fixture"}, nil
}

type fakeSkill struct{ calls int }

func (f *fakeSkill) Invoke(context.Context, skilldto.InvokeInput) (*skilldto.InvokeOutput, error) {
	f.calls++
	return &skilldto.InvokeOutput{Status: "completed", Result: map[string]any{"count": 1}}, nil
}
func TestLocalManagementPersistenceIsolationAndExecution(t *testing.T) {
	models.InitSchemaFrom("")
	path := filepath.Join(t.TempDir(), "local.db")
	db, e := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.Agent{}, &model.Skill{}); e != nil {
		t.Fatal(e)
	}
	ai, exec := &fakeAI{}, &fakeSkill{}
	s := New(db, ai, exec)
	ctx := WithActor(authx.ContextWithTenantUUID(context.Background(), uuid.NewString()), uuid.NewString())
	other := authx.ContextWithTenantUUID(context.Background(), uuid.NewString())
	input := skillfw.DefinitionInput{Key: "test.skill", Name: "fixture", Version: "1.0.0", Executor: "template", Status: "active"}
	skill, e := s.SaveSkill(ctx, "", input)
	if e != nil {
		t.Fatal(e)
	}
	agentIn := agentfw.DefinitionInput{Key: "test.agent", Name: "fixture", Status: "active", ModelKey: "test.model", SkillUUIDs: []string{skill.UUID}, Persona: "persona", Prompt: "prompt"}
	agent, e := s.SaveAgent(ctx, "", agentIn)
	if e != nil {
		t.Fatal(e)
	}
	output, e := s.DebugAgent(ctx, agent.UUID, agentfw.DebugInput{Message: "fixture", SkillUUID: skill.UUID, SkillInput: map[string]any{"action": "list"}})
	if e != nil || output.SkillResult == nil || exec.calls != 1 || ai.calls != 1 || len(ai.last.Inputs) != 3 {
		t.Fatalf("execution: %+v %v calls=%d/%d", output, e, exec.calls, ai.calls)
	}
	// Reopening the store retains definitions and UUID references.
	reopened, e := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	s = New(reopened, ai, exec)
	rows, e := s.ListAgents(ctx, skillfw.DefinitionQuery{})
	if e != nil || rows.Total != 1 || rows.Items[0].SkillUUIDs[0] != skill.UUID {
		t.Fatalf("persistence: %+v %v", rows, e)
	}
	empty, e := s.ListAgents(other, skillfw.DefinitionQuery{})
	if e != nil || empty.Total != 0 {
		t.Fatal(empty, e)
	}
	_, e = s.SaveAgent(other, agent.UUID, agentIn)
	status, _ := ErrorStatus(e)
	if status != 404 {
		t.Fatalf("cross tenant update %v", e)
	}
	_, e = s.SaveAgent(other, "", agentIn)
	status, _ = ErrorStatus(e)
	if status != 404 {
		t.Fatalf("cross tenant binding %v", e)
	}
	_, e = s.InvokeSkill(other, skill.UUID, nil)
	status, _ = ErrorStatus(e)
	if status != 404 {
		t.Fatalf("cross tenant invoke %v", e)
	}
	_, e = s.InvokeSkill(context.Background(), skill.UUID, nil)
	status, _ = ErrorStatus(e)
	if status != 401 {
		t.Fatalf("missing tenant %v", e)
	}
	_, e = s.SaveSkill(ctx, "", input)
	status, _ = ErrorStatus(e)
	if status != 409 {
		t.Fatalf("duplicate %v", e)
	}
	_, e = s.DebugAgent(ctx, agent.UUID, agentfw.DebugInput{Message: "fixture", SkillUUID: uuid.NewString()})
	status, _ = ErrorStatus(e)
	if status != 403 {
		t.Fatalf("unbound %v", e)
	}
	input.Status = "inactive"
	_, e = s.SaveSkill(ctx, skill.UUID, input)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DebugAgent(ctx, agent.UUID, agentfw.DebugInput{Message: "fixture", SkillUUID: skill.UUID})
	status, _ = ErrorStatus(e)
	if status != 409 || exec.calls != 1 || ai.calls != 1 {
		t.Fatalf("disabled skill executed: %v", e)
	}
	agentIn.Status = "inactive"
	agentIn.Name = "updated"
	_, e = s.SaveAgent(ctx, agent.UUID, agentIn)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DebugAgent(ctx, agent.UUID, agentfw.DebugInput{Message: "fixture"})
	status, _ = ErrorStatus(e)
	if status != 409 {
		t.Fatalf("disabled agent %v", e)
	}
	rows, e = s.ListAgents(ctx, skillfw.DefinitionQuery{Query: "updated", Status: "inactive"})
	if e != nil || rows.Total != 1 {
		t.Fatal(rows, e)
	}
}
func TestPromptSkillRequiresConfiguredModel(t *testing.T) {
	models.InitSchemaFrom("")
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.Skill{}); e != nil {
		t.Fatal(e)
	}
	ai := &fakeAI{}
	s := New(db, ai, nil)
	ctx := WithActor(authx.ContextWithTenantUUID(context.Background(), uuid.NewString()), uuid.NewString())
	input := skillfw.DefinitionInput{Key: "prompt", Name: "fixture", Version: "1.0.0", Executor: "prompt", Status: "active", ModelKey: "missing", Prompt: "fixture"}
	if _, e = s.SaveSkill(ctx, "", input); e == nil {
		t.Fatal("accepted missing model")
	}
	input.ModelKey = "test.model"
	item, e := s.SaveSkill(ctx, "", input)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.InvokeSkill(ctx, item.UUID, map[string]any{"value": 1}); e != nil || ai.calls != 1 {
		t.Fatal(e)
	}
	if ai.last.Inputs[0].Content != input.Prompt {
		t.Fatal("prompt not applied")
	}
}

func TestRealLocalTemplateExecutor(t *testing.T) {
	models.InitSchemaFrom("")
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.Skill{}, &templatemodel.Template{}); e != nil {
		t.Fatal(e)
	}
	reg, e := builtinskills.NewTemplateRegistry(db)
	if e != nil {
		t.Fatal(e)
	}
	invoker := builtinskills.NewFrameworkLocalInvoker(reg, authx.TenantUUIDFromContext, "com.powerx.test")
	s := New(db, nil, invoker)
	ctx := WithActor(authx.ContextWithTenantUUID(context.Background(), uuid.NewString()), uuid.NewString())
	def, e := s.SaveSkill(ctx, "", skillfw.DefinitionInput{Key: "template", Name: "fixture", Version: "1.0.0", Executor: "template", Status: "active"})
	if e != nil {
		t.Fatal(e)
	}
	out, e := s.InvokeSkill(ctx, def.UUID, map[string]any{"action": "list"})
	if e != nil {
		t.Fatal(e)
	}
	if out["status"] != "completed" {
		t.Fatalf("result %v", out)
	}
}
