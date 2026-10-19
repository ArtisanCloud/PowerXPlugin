package regression

import (
	"encoding/json"
	Event "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"time"
)

type EventMeta = Event.Meta

type SuiteDefinition struct {
	UUID           string `json:"uuid"`
	TitleKey       string `json:"title_key"`
	DescriptionKey string `json:"description_key"`
	Key            string `json:"key"`
}

type StepDefinition struct {
	UUID     string `json:"uuid"`
	Key      string `json:"key"`
	TitleKey string `json:"title_key"`
	Phase    string `json:"phase"`
	Boundary string `json:"boundary"`
}

type CaseDefinition struct {
	UUID                  string           `json:"uuid"`
	SuiteUUID             string           `json:"suite_uuid"`
	Key                   string           `json:"key"`
	Version               string           `json:"version"`
	TitleKey              string           `json:"title_key"`
	DescriptionKey        string           `json:"description_key"`
	Tags                  []string         `json:"tags"`
	SupportedProfileKinds []string         `json:"supported_profile_kinds"`
	InputSchema           json.RawMessage  `json:"input_schema"`
	ExecutionTimeoutMS    uint64           `json:"execution_timeout_ms"`
	CleanupTimeoutMS      uint64           `json:"cleanup_timeout_ms"`
	Steps                 []StepDefinition `json:"steps"`
	RequiredAssertionKeys []string         `json:"required_assertion_keys"`
}

type ProfileDefinition struct {
	UUID                   string                  `json:"uuid"`
	Key                    string                  `json:"key"`
	TitleKey               string                  `json:"title_key"`
	Kind                   string                  `json:"kind"`
	DependencyDeclarations []DependencyDeclaration `json:"dependency_declarations"`
	FixtureNamespace       string                  `json:"fixture_namespace"`
	BindingRevision        string                  `json:"binding_revision"`
}

type DependencyDeclaration struct {
	DependencyKey        string   `json:"dependency_key"`
	Mode                 string   `json:"mode"`
	Boundary             string   `json:"boundary"`
	RequiredCapabilities []string `json:"required_capabilities"`
}

type DependencyEvidence struct {
	DependencyKey string `json:"dependency_key"`
	DeclaredMode  string `json:"declared_mode"`
	ActualMode    string `json:"actual_mode"`
	Boundary      string `json:"boundary"`
	ReasonCode    string `json:"reason_code"`
}

type Evidence struct {
	Kind          string          `json:"kind"`
	Boundary      string          `json:"boundary"`
	SourceRef     string          `json:"source_ref"`
	ObservedAt    time.Time       `json:"observed_at"`
	SanitizedData json.RawMessage `json:"sanitized_data"`
	ResourceUUID  string          `json:"resource_uuid"`
	TraceID       string          `json:"trace_id"`
	RequestID     string          `json:"request_id"`
}

type AssertionResult struct {
	UUID       string          `json:"uuid"`
	Key        string          `json:"key"`
	TitleKey   string          `json:"title_key"`
	Status     string          `json:"status"`
	Required   bool            `json:"required"`
	Expected   json.RawMessage `json:"expected"`
	Actual     json.RawMessage `json:"actual"`
	Evidence   []Evidence      `json:"evidence"`
	ReasonCode string          `json:"reason_code"`
}

type StepResult struct {
	UUID       string            `json:"uuid"`
	StepUUID   string            `json:"step_uuid"`
	Phase      string            `json:"phase"`
	TitleKey   string            `json:"title_key"`
	Status     string            `json:"status"`
	StartedAt  *time.Time        `json:"started_at"`
	FinishedAt *time.Time        `json:"finished_at"`
	Assertions []AssertionResult `json:"assertions"`
}

type FixtureResource struct {
	UUID               string          `json:"uuid"`
	ResourceType       string          `json:"resource_type"`
	ResourceUUID       string          `json:"resource_uuid"`
	DisplayName        string          `json:"display_name"`
	RoleKey            string          `json:"role_key"`
	ParentResourceUUID string          `json:"parent_resource_uuid"`
	TenantUUID         string          `json:"tenant_uuid"`
	RunUUID            string          `json:"run_uuid"`
	CaseExecutionUUID  string          `json:"case_execution_uuid"`
	CleanupKey         string          `json:"cleanup_key"`
	CleanupParameters  json.RawMessage `json:"cleanup_parameters"`
	CleanupState       string          `json:"cleanup_state"`
	CleanupEvidence    []Evidence      `json:"cleanup_evidence"`
	CleanedAt          *time.Time      `json:"cleaned_at"`
}

type Run struct {
	UUID              string            `json:"uuid"`
	TenantUUID        string            `json:"tenant_uuid"`
	ActorUUID         string            `json:"actor_uuid"`
	ProfileUUID       string            `json:"profile_uuid"`
	ClientRequestUUID string            `json:"client_request_uuid"`
	RerunOfRunUUID    *string           `json:"rerun_of_run_uuid"`
	ProfileSnapshot   ProfileDefinition `json:"profile_snapshot"`
	SelectionSnapshot []CaseDefinition  `json:"selection_snapshot"`
	InputSnapshot     json.RawMessage   `json:"input_snapshot"`
	SourceSnapshot    json.RawMessage   `json:"source_snapshot"`
	Phase             string            `json:"phase"`
	Verdict           string            `json:"verdict"`
	CleanupState      string            `json:"cleanup_state"`
	ReasonCode        string            `json:"reason_code"`
	MessageKey        string            `json:"message_key"`
	Revision          uint64            `json:"revision"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	StartedAt         *time.Time        `json:"started_at"`
	FinishedAt        *time.Time        `json:"finished_at"`
}

type CaseExecution struct {
	UUID                 string               `json:"uuid"`
	TenantUUID           string               `json:"tenant_uuid"`
	RunUUID              string               `json:"run_uuid"`
	CaseUUID             string               `json:"case_uuid"`
	Ordinal              uint64               `json:"ordinal"`
	DefinitionSnapshot   CaseDefinition       `json:"definition_snapshot"`
	InputSnapshot        json.RawMessage      `json:"input_snapshot"`
	DependencyEvidence   []DependencyEvidence `json:"dependency_evidence"`
	Phase                string               `json:"phase"`
	Verdict              string               `json:"verdict"`
	Steps                []StepResult         `json:"steps"`
	FixtureResources     []FixtureResource    `json:"fixture_resources"`
	CleanupState         string               `json:"cleanup_state"`
	ReasonCode           string               `json:"reason_code"`
	MessageKey           string               `json:"message_key"`
	Revision             uint64               `json:"revision"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	StartedAt            *time.Time           `json:"started_at"`
	FinishedAt           *time.Time           `json:"finished_at"`
	MissingAssertionKeys []string             `json:"missing_assertion_keys"`
}

type RegressionEvent struct {
	UUID              string          `json:"uuid"`
	TenantUUID        string          `json:"tenant_uuid"`
	RunUUID           string          `json:"run_uuid"`
	CaseExecutionUUID *string         `json:"case_execution_uuid"`
	Sequence          uint64          `json:"sequence"`
	Revision          uint64          `json:"revision"`
	Kind              string          `json:"kind"`
	Payload           json.RawMessage `json:"payload"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Meta              EventMeta       `json:"meta"`
}

type RunDetail struct {
	RequestHash    string          `json:"-"`
	Run            Run             `json:"run"`
	CaseExecutions []CaseExecution `json:"case_executions"`
	LastEventUUID  *string         `json:"last_event_uuid"`
}

type Report struct {
	SchemaVersion  string            `json:"schema_version"`
	Run            Run               `json:"run"`
	CaseExecutions []CaseExecution   `json:"case_executions"`
	Events         []RegressionEvent `json:"events"`
	ExportedAt     time.Time         `json:"exported_at"`
}

type CreateRun struct {
	ClientRequestUUID string          `json:"client_request_uuid"`
	ProfileUUID       string          `json:"profile_uuid"`
	CaseUuids         []string        `json:"case_uuids"`
	Inputs            json.RawMessage `json:"inputs"`
	RerunOfRunUUID    string          `json:"rerun_of_run_uuid"`
}
