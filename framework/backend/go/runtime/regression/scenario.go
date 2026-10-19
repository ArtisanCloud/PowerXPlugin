package regression

import (
	"context"
	"encoding/json"
)

type Actor struct {
	TenantUUID string
	UUID       string
	MemberUUID string
	PluginID   string
	RequestID  string
	TraceID    string
	Credential string
}
type Scenario interface {
	Definition() CaseDefinition
	Preflight(context.Context, *Scope) error
	Prepare(context.Context, *Scope) error
	Execute(context.Context, *Scope) error
	Cleanup(context.Context, *Scope) error
}
type ScenarioFactory func() Scenario
type ProfileFactory func(context.Context, Actor) (any, []DependencyEvidence, error)
type RecordPort interface {
	Step(context.Context, string, func(context.Context) error) error
	Assert(context.Context, string, string, string, any, any, []Evidence) error
	Register(context.Context, FixtureResource) error
	ListOwned() []FixtureResource
	Cleaned(context.Context, string, []Evidence) error
}
type Scope struct {
	RunUUID           string
	CaseExecutionUUID string
	Actor             Actor
	Definition        CaseDefinition
	Profile           ProfileDefinition
	Input             json.RawMessage
	Dependencies      any
	Recorder          RecordPort
}
