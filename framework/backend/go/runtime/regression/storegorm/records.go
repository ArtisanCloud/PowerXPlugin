package storegorm

import (
	"encoding/json"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"time"
)

type RunRecord struct {
	UUID              string                       `gorm:"column:uuid;type:uuid;primaryKey;not null" json:"uuid"`
	TenantUUID        string                       `gorm:"column:tenant_uuid;type:uuid;index;not null" json:"tenant_uuid"`
	ActorUUID         string                       `gorm:"column:actor_uuid;type:uuid;index;not null" json:"actor_uuid"`
	ProfileUUID       string                       `gorm:"column:profile_uuid;type:uuid;index;not null" json:"profile_uuid"`
	ClientRequestUUID string                       `gorm:"column:client_request_uuid;type:uuid;index;not null" json:"client_request_uuid"`
	RerunOfRunUUID    *string                      `gorm:"column:rerun_of_run_uuid;type:uuid;index" json:"rerun_of_run_uuid"`
	ProfileSnapshot   Regression.ProfileDefinition `gorm:"column:profile_snapshot;serializer:json;type:jsonb;not null" json:"profile_snapshot"`
	SelectionSnapshot []Regression.CaseDefinition  `gorm:"column:selection_snapshot;serializer:json;type:jsonb;not null" json:"selection_snapshot"`
	InputSnapshot     json.RawMessage              `gorm:"column:input_snapshot;serializer:json;type:jsonb;not null" json:"input_snapshot"`
	SourceSnapshot    json.RawMessage              `gorm:"column:source_snapshot;serializer:json;type:jsonb;not null" json:"source_snapshot"`
	Phase             string                       `gorm:"column:phase;not null" json:"phase"`
	Verdict           string                       `gorm:"column:verdict;not null" json:"verdict"`
	CleanupState      string                       `gorm:"column:cleanup_state;not null" json:"cleanup_state"`
	ReasonCode        string                       `gorm:"column:reason_code;not null" json:"reason_code"`
	MessageKey        string                       `gorm:"column:message_key;not null" json:"message_key"`
	Revision          uint64                       `gorm:"column:revision;not null" json:"revision"`
	CreatedAt         time.Time                    `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt         time.Time                    `gorm:"column:updated_at;not null" json:"updated_at"`
	StartedAt         *time.Time                   `gorm:"column:started_at" json:"started_at"`
	FinishedAt        *time.Time                   `gorm:"column:finished_at" json:"finished_at"`
	RequestHash       string                       `gorm:"column:request_hash;type:varchar(64);not null" json:"-"`
}

type CaseExecutionRecord struct {
	UUID                 string                          `gorm:"column:uuid;type:uuid;primaryKey;not null" json:"uuid"`
	TenantUUID           string                          `gorm:"column:tenant_uuid;type:uuid;index;not null" json:"tenant_uuid"`
	RunUUID              string                          `gorm:"column:run_uuid;type:uuid;index;not null" json:"run_uuid"`
	CaseUUID             string                          `gorm:"column:case_uuid;type:uuid;index;not null" json:"case_uuid"`
	Ordinal              uint64                          `gorm:"column:ordinal;not null" json:"ordinal"`
	DefinitionSnapshot   Regression.CaseDefinition       `gorm:"column:definition_snapshot;serializer:json;type:jsonb;not null" json:"definition_snapshot"`
	InputSnapshot        json.RawMessage                 `gorm:"column:input_snapshot;serializer:json;type:jsonb;not null" json:"input_snapshot"`
	DependencyEvidence   []Regression.DependencyEvidence `gorm:"column:dependency_evidence;serializer:json;type:jsonb;not null" json:"dependency_evidence"`
	Phase                string                          `gorm:"column:phase;not null" json:"phase"`
	Verdict              string                          `gorm:"column:verdict;not null" json:"verdict"`
	Steps                []Regression.StepResult         `gorm:"column:steps;serializer:json;type:jsonb;not null" json:"steps"`
	FixtureResources     []Regression.FixtureResource    `gorm:"column:fixture_resources;serializer:json;type:jsonb;not null" json:"fixture_resources"`
	CleanupState         string                          `gorm:"column:cleanup_state;not null" json:"cleanup_state"`
	ReasonCode           string                          `gorm:"column:reason_code;not null" json:"reason_code"`
	MessageKey           string                          `gorm:"column:message_key;not null" json:"message_key"`
	Revision             uint64                          `gorm:"column:revision;not null" json:"revision"`
	CreatedAt            time.Time                       `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt            time.Time                       `gorm:"column:updated_at;not null" json:"updated_at"`
	StartedAt            *time.Time                      `gorm:"column:started_at" json:"started_at"`
	FinishedAt           *time.Time                      `gorm:"column:finished_at" json:"finished_at"`
	MissingAssertionKeys []string                        `gorm:"column:missing_assertion_keys;serializer:json;type:jsonb;not null" json:"missing_assertion_keys"`
}

type RegressionEventRecord struct {
	UUID              string               `gorm:"column:uuid;type:uuid;primaryKey;not null" json:"uuid"`
	TenantUUID        string               `gorm:"column:tenant_uuid;type:uuid;index;not null" json:"tenant_uuid"`
	RunUUID           string               `gorm:"column:run_uuid;type:uuid;index;not null" json:"run_uuid"`
	CaseExecutionUUID *string              `gorm:"column:case_execution_uuid;type:uuid;index" json:"case_execution_uuid"`
	Sequence          uint64               `gorm:"column:sequence;not null" json:"sequence"`
	Revision          uint64               `gorm:"column:revision;not null" json:"revision"`
	Kind              string               `gorm:"column:kind;not null" json:"kind"`
	Payload           json.RawMessage      `gorm:"column:payload;serializer:json;type:jsonb;not null" json:"payload"`
	CreatedAt         time.Time            `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt         time.Time            `gorm:"column:updated_at;not null" json:"updated_at"`
	Meta              Regression.EventMeta `gorm:"column:meta;serializer:json;type:jsonb;not null" json:"meta"`
}
