package skills

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type Definition struct {
	UUID        string    `json:"uuid"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	Executor    string    `json:"executor"`
	ModelKey    string    `json:"model_key"`
	Prompt      string    `json:"prompt"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type DefinitionInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Executor    string `json:"executor"`
	ModelKey    string `json:"model_key"`
	Prompt      string `json:"prompt"`
}
type DefinitionQuery struct {
	Query, Status  string
	Page, PageSize int
}
type DefinitionPage struct {
	Items    []Definition `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}
type ManagementService interface {
	ListSkills(context.Context, DefinitionQuery) (*DefinitionPage, error)
	SaveSkill(context.Context, string, DefinitionInput) (*Definition, error)
	InvokeSkill(context.Context, string, map[string]any) (map[string]any, error)
}

func NewManagementRuntime(mode provider.Mode, local, delegated ManagementService) (*module.Factory[ManagementService], error) {
	return module.NewFactory("skills.management", mode,
		module.Binding[ManagementService]{Value: local, Available: local != nil},
		module.Binding[ManagementService]{Value: delegated, Available: delegated != nil})
}
