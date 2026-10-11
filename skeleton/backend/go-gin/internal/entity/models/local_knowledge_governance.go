package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// The records below mirror the Core knowledge domain.  They deliberately live
// in plugin-owned local_knowledge_* tables; cross-record references are UUIDs,
// never Core's internal numeric IDs.

type LocalKnowledgeArtifactBundle struct {
	BaseNoTenantModel
	UUID                string     `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	IngestionJobUUID    string     `gorm:"type:uuid;not null;uniqueIndex" json:"ingestion_job_uuid"`
	ChunkManifestURI    string     `gorm:"type:text;not null" json:"chunk_manifest_uri"`
	VectorManifestURI   string     `gorm:"type:text" json:"vector_manifest_uri,omitempty"`
	GraphManifestURI    string     `gorm:"type:text" json:"graph_manifest_uri,omitempty"`
	MaskingReportURI    string     `gorm:"type:text" json:"masking_report_uri,omitempty"`
	OCRManifestURI      string     `gorm:"type:text" json:"ocr_manifest_uri,omitempty"`
	SummaryChunkCount   int        `gorm:"not null;default:0" json:"summary_chunk_count"`
	ParagraphChunkCount int        `gorm:"not null;default:0" json:"paragraph_chunk_count"`
	Checksum            string     `gorm:"type:char(64);not null" json:"checksum"`
	StorageClass        string     `gorm:"type:varchar(32);not null;default:standard" json:"storage_class"`
	Status              string     `gorm:"type:varchar(32);not null;default:active;index" json:"status"`
	RetainedUntil       *time.Time `json:"retained_until,omitempty"`
}

func (LocalKnowledgeArtifactBundle) TableName() string { return S(TableLocalKnowledgeArtifactBundles) }
func (m *LocalKnowledgeArtifactBundle) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeAuditTrailEntry struct {
	BaseNoTenantModel
	UUID          string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID     string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	Action        string         `gorm:"type:varchar(64);not null" json:"action"`
	Actor         string         `gorm:"type:varchar(128);not null" json:"actor"`
	PayloadHash   string         `gorm:"type:char(64);not null" json:"payload_hash"`
	Metadata      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	OccurredAt    time.Time      `gorm:"not null;index" json:"occurred_at"`
	RollbackToken string         `gorm:"type:varchar(128)" json:"rollback_token,omitempty"`
}

func (LocalKnowledgeAuditTrailEntry) TableName() string { return S(TableLocalKnowledgeAuditTrail) }
func (m *LocalKnowledgeAuditTrailEntry) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeCorpusCheckJob struct {
	BaseNoTenantModel
	UUID            string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID      string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_corpus_check_tenant_space,priority:1;index" json:"tenant_uuid"`
	SpaceUUID       string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_corpus_check_tenant_space,priority:2;index" json:"space_uuid"`
	Status          string         `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	SampleJobUUIDs  datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"sample_job_uuids"`
	Metrics         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metrics"`
	Recommendations datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"recommendations"`
	TraceID         string         `gorm:"type:varchar(128);index" json:"trace_id,omitempty"`
	ErrorReason     string         `gorm:"type:text" json:"error_reason,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

func (LocalKnowledgeCorpusCheckJob) TableName() string { return S(TableLocalKnowledgeCorpusChecks) }
func (m *LocalKnowledgeCorpusCheckJob) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeDecayTask struct {
	BaseNoTenantModel
	UUID       string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID  string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	Category   string         `gorm:"type:varchar(64);not null" json:"category"`
	Severity   string         `gorm:"type:varchar(16);not null;default:medium" json:"severity"`
	Status     string         `gorm:"type:varchar(32);not null;default:open;index" json:"status"`
	DetectedAt time.Time      `gorm:"not null" json:"detected_at"`
	SLADueAt   time.Time      `gorm:"not null" json:"sla_due_at"`
	ResolvedAt *time.Time     `json:"resolved_at,omitempty"`
	Owner      string         `gorm:"type:varchar(128)" json:"owner,omitempty"`
	Details    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
}

func (LocalKnowledgeDecayTask) TableName() string { return S(TableLocalKnowledgeDecayTasks) }
func (m *LocalKnowledgeDecayTask) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeDeltaJob struct {
	BaseNoTenantModel
	UUID           string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID      string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	Source         string         `gorm:"type:varchar(128);not null" json:"source"`
	PackageURI     string         `gorm:"type:text;not null" json:"package_uri"`
	Status         string         `gorm:"type:varchar(32);not null;default:generated;index" json:"status"`
	ApprovalState  string         `gorm:"type:varchar(32);not null;default:pending" json:"approval_state"`
	DiffAccuracy   float64        `gorm:"not null;default:0" json:"diff_accuracy"`
	PartialRelease bool           `gorm:"not null;default:false" json:"partial_release"`
	ApprovedBy     string         `gorm:"type:varchar(128)" json:"approved_by,omitempty"`
	ApprovedAt     *time.Time     `json:"approved_at,omitempty"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	RollbackCount  int            `gorm:"not null;default:0" json:"rollback_count"`
	Report         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"report"`
	Notes          string         `gorm:"type:text" json:"notes,omitempty"`
}

func (LocalKnowledgeDeltaJob) TableName() string { return S(TableLocalKnowledgeDeltaJobs) }
func (m *LocalKnowledgeDeltaJob) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeFeedbackCase struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID        string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	ReportedBy       string         `gorm:"type:varchar(128);not null" json:"reported_by"`
	IssueType        string         `gorm:"type:varchar(64);not null" json:"issue_type"`
	Severity         string         `gorm:"type:varchar(16);not null;default:medium" json:"severity"`
	Status           string         `gorm:"type:varchar(32);not null;default:open;index" json:"status"`
	LinkedChunks     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"linked_chunks"`
	ToolTrace        string         `gorm:"type:varchar(128)" json:"tool_trace,omitempty"`
	QualityScore     float64        `gorm:"not null;default:0" json:"quality_score"`
	SLADueAt         *time.Time     `json:"sla_due_at,omitempty"`
	Resolution       string         `gorm:"type:text" json:"resolution,omitempty"`
	ReprocessJobUUID *string        `gorm:"type:uuid" json:"reprocess_job_uuid,omitempty"`
	ResolvedAt       *time.Time     `json:"resolved_at,omitempty"`
}

func (LocalKnowledgeFeedbackCase) TableName() string { return S(TableLocalKnowledgeFeedbackCases) }
func (m *LocalKnowledgeFeedbackCase) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeFusionStrategyVersion struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID        string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	Label            string         `gorm:"type:varchar(128);not null" json:"label"`
	BM25Weight       float64        `gorm:"not null;default:0" json:"bm25_weight"`
	VectorWeight     float64        `gorm:"not null;default:0" json:"vector_weight"`
	GraphConstraint  string         `gorm:"type:text" json:"graph_constraint,omitempty"`
	RerankerModel    string         `gorm:"type:varchar(128)" json:"reranker_model,omitempty"`
	ConflictPolicy   string         `gorm:"type:varchar(64);not null;default:prefer_vector" json:"conflict_policy"`
	DeploymentState  string         `gorm:"type:varchar(32);not null;default:draft;index" json:"deployment_state"`
	BenchmarkMetrics datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"benchmark_metrics"`
	PublishedAt      *time.Time     `json:"published_at,omitempty"`
	RollbackFromUUID *string        `gorm:"type:uuid" json:"rollback_from_uuid,omitempty"`
}

func (LocalKnowledgeFusionStrategyVersion) TableName() string {
	return S(TableLocalKnowledgeFusionStrategies)
}
func (m *LocalKnowledgeFusionStrategyVersion) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeIAMSyncTask struct {
	BaseNoTenantModel
	UUID        string     `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	SpaceUUID   string     `gorm:"type:uuid;not null;index" json:"space_uuid"`
	Provider    string     `gorm:"type:varchar(64);not null" json:"provider"`
	Status      string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	RetryCount  int        `gorm:"not null;default:0" json:"retry_count"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (LocalKnowledgeIAMSyncTask) TableName() string { return S(TableLocalKnowledgeIAMSyncTasks) }
func (m *LocalKnowledgeIAMSyncTask) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgePolicyTemplateVersion struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TemplateName     string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_local_policy_template_version,priority:1" json:"template_name"`
	Version          int            `gorm:"not null;uniqueIndex:uk_local_policy_template_version,priority:2" json:"version"`
	RAGProfile       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"rag_profile"`
	GraphSettings    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"graph_settings"`
	MaskingSettings  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"masking_settings"`
	AlertingSettings datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"alerting_settings"`
	ApprovedBy       string         `gorm:"type:varchar(128)" json:"approved_by,omitempty"`
	ApprovedAt       *time.Time     `json:"approved_at,omitempty"`
	RollbackToken    string         `gorm:"type:varchar(128)" json:"rollback_token,omitempty"`
	ImmutableHash    string         `gorm:"type:char(64);not null;uniqueIndex" json:"immutable_hash"`
}

func (LocalKnowledgePolicyTemplateVersion) TableName() string {
	return S(TableLocalKnowledgePolicyTemplates)
}
func (m *LocalKnowledgePolicyTemplateVersion) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeSourceConnectorInstance struct {
	BaseNoTenantModel
	UUID           string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID     string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	Provider       string         `gorm:"type:varchar(64);not null" json:"provider"`
	CredentialUUID string         `gorm:"type:uuid;not null;index" json:"credential_uuid"`
	Status         string         `gorm:"type:varchar(32);not null;default:active;index" json:"status"`
	Config         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	LastError      string         `gorm:"type:text" json:"last_error,omitempty"`
	CreatedBy      string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
	UpdatedBy      string         `gorm:"type:varchar(128)" json:"updated_by,omitempty"`
}

func (LocalKnowledgeSourceConnectorInstance) TableName() string {
	return S(TableLocalKnowledgeSourceConnectors)
}
func (m *LocalKnowledgeSourceConnectorInstance) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeSourceCredential struct {
	BaseNoTenantModel
	UUID       string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	Provider   string         `gorm:"type:varchar(64);not null" json:"provider"`
	AuthType   string         `gorm:"type:varchar(32);not null" json:"auth_type"`
	Label      string         `gorm:"type:varchar(128);not null" json:"label"`
	Status     string         `gorm:"type:varchar(32);not null;default:active;index" json:"status"`
	SecretRef  string         `gorm:"type:text;not null" json:"secret_ref"`
	Metadata   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedBy  string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
	UpdatedBy  string         `gorm:"type:varchar(128)" json:"updated_by,omitempty"`
}

func (LocalKnowledgeSourceCredential) TableName() string {
	return S(TableLocalKnowledgeSourceCredentials)
}
func (m *LocalKnowledgeSourceCredential) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeSpaceSyncJob struct {
	BaseNoTenantModel
	UUID          string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID    string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID     string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	ConnectorUUID string         `gorm:"type:uuid;not null;index" json:"connector_uuid"`
	Provider      string         `gorm:"type:varchar(64);not null" json:"provider"`
	SyncMode      string         `gorm:"type:varchar(32);not null" json:"sync_mode"`
	Schedule      string         `gorm:"type:varchar(128)" json:"schedule,omitempty"`
	Status        string         `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	Scope         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"scope"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	LastError     string         `gorm:"type:text" json:"last_error,omitempty"`
	ExternalRef   string         `gorm:"type:varchar(256)" json:"external_ref,omitempty"`
	CreatedBy     string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
	UpdatedBy     string         `gorm:"type:varchar(128)" json:"updated_by,omitempty"`
}

func (LocalKnowledgeSpaceSyncJob) TableName() string { return S(TableLocalKnowledgeSpaceSyncJobs) }
func (m *LocalKnowledgeSpaceSyncJob) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeReleasePolicy struct {
	BaseNoTenantModel
	UUID          string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	MatrixVersion string         `gorm:"type:varchar(128);not null" json:"matrix_version"`
	PilotTenants  datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"pilot_tenants"`
	Batches       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"batches"`
	Guardrails    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"guardrails"`
	Status        string         `gorm:"type:varchar(32);not null;default:draft;index" json:"status"`
	ApprovedBy    string         `gorm:"type:varchar(128)" json:"approved_by,omitempty"`
	CreatedBy     string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
}

func (LocalKnowledgeReleasePolicy) TableName() string { return S(TableLocalKnowledgeReleasePolicies) }
func (m *LocalKnowledgeReleasePolicy) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

type LocalKnowledgeReleaseBatch struct {
	BaseNoTenantModel
	UUID        string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	PolicyUUID  string         `gorm:"type:uuid;not null;index" json:"policy_uuid"`
	VersionID   string         `gorm:"type:varchar(128);not null" json:"version_id"`
	BatchIndex  int            `gorm:"not null" json:"batch_index"`
	Tenants     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tenants"`
	State       string         `gorm:"type:varchar(32);not null;default:pending;index" json:"state"`
	Alerts      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"alerts"`
	Metrics     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metrics"`
	BatchToken  string         `gorm:"type:varchar(128);not null;uniqueIndex" json:"batch_token"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

func (LocalKnowledgeReleaseBatch) TableName() string { return S(TableLocalKnowledgeReleaseBatches) }
func (m *LocalKnowledgeReleaseBatch) BeforeCreate(*gorm.DB) error {
	ensureLocalKnowledgeUUID(&m.UUID)
	return nil
}

func ensureLocalKnowledgeUUID(value *string) {
	if strings.TrimSpace(*value) == "" {
		*value = uuid.NewString()
	}
}
