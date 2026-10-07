package agent

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	"time"
)

type Definition struct {
	UUID        string    `json:"uuid"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ModelKey    string    `json:"model_key"`
	Persona     string    `json:"persona"`
	Prompt      string    `json:"prompt"`
	SkillUUIDs  []string  `json:"skill_uuids"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type DefinitionInput struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	ModelKey    string   `json:"model_key"`
	Persona     string   `json:"persona"`
	Prompt      string   `json:"prompt"`
	SkillUUIDs  []string `json:"skill_uuids"`
}
type DefinitionPage struct {
	Items    []Definition `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}
type DebugInput struct {
	Message    string         `json:"message"`
	SkillUUID  string         `json:"skill_uuid,omitempty"`
	SkillInput map[string]any `json:"skill_input,omitempty"`
}
type DebugOutput struct {
	Text        string         `json:"text"`
	TraceID     string         `json:"trace_id"`
	SkillResult map[string]any `json:"skill_result,omitempty"`
}
type ManagementService interface {
	ListAgents(context.Context, skills.DefinitionQuery) (*DefinitionPage, error)
	SaveAgent(context.Context, string, DefinitionInput) (*Definition, error)
	DebugAgent(context.Context, string, DebugInput) (*DebugOutput, error)
}

func NewManagementRuntime(mode provider.Mode, local, delegated ManagementService) (*module.Factory[ManagementService], error) {
	return module.NewFactory("agent.management", mode,
		module.Binding[ManagementService]{Value: local, Available: local != nil},
		module.Binding[ManagementService]{Value: delegated, Available: delegated != nil})
}
