package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Local* records keep the same domain meaning as the Core knowledge models,
// while staying in plugin-owned tables and using UUID relations only.
type LocalIngestionProfileVersion struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID       string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_ing_profile,priority:1;index" json:"tenant_uuid"`
	ProfileKey       string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_local_ing_profile,priority:2" json:"profile_key"`
	Version          int            `gorm:"not null;uniqueIndex:uk_local_ing_profile,priority:3" json:"version"`
	Status           string         `gorm:"type:varchar(32);not null;default:draft;index" json:"status"`
	DisplayName      string         `gorm:"type:varchar(128);not null" json:"display_name"`
	Config           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	RollbackFromUUID *string        `gorm:"type:uuid" json:"rollback_from_uuid,omitempty"`
	PublishedAt      *time.Time     `json:"published_at,omitempty"`
	PublishedBy      string         `gorm:"type:varchar(128)" json:"published_by,omitempty"`
	CreatedBy        string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
}

func (LocalIngestionProfileVersion) TableName() string {
	return S(TableLocalKnowledgeIngestionProfiles)
}
func (m *LocalIngestionProfileVersion) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalIndexProfileVersion struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID       string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_index_profile,priority:1;index" json:"tenant_uuid"`
	ProfileKey       string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_local_index_profile,priority:2" json:"profile_key"`
	Version          int            `gorm:"not null;uniqueIndex:uk_local_index_profile,priority:3" json:"version"`
	Status           string         `gorm:"type:varchar(32);not null;default:draft;index" json:"status"`
	DisplayName      string         `gorm:"type:varchar(128);not null" json:"display_name"`
	Config           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	RollbackFromUUID *string        `gorm:"type:uuid" json:"rollback_from_uuid,omitempty"`
	PublishedAt      *time.Time     `json:"published_at,omitempty"`
	PublishedBy      string         `gorm:"type:varchar(128)" json:"published_by,omitempty"`
	CreatedBy        string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
}

func (LocalIndexProfileVersion) TableName() string { return S(TableLocalKnowledgeIndexProfiles) }
func (m *LocalIndexProfileVersion) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalRAGProfileVersion struct {
	BaseNoTenantModel
	UUID             string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID       string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_rag_profile,priority:1;index" json:"tenant_uuid"`
	ProfileKey       string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_local_rag_profile,priority:2" json:"profile_key"`
	Version          int            `gorm:"not null;uniqueIndex:uk_local_rag_profile,priority:3" json:"version"`
	Status           string         `gorm:"type:varchar(32);not null;default:draft;index" json:"status"`
	DisplayName      string         `gorm:"type:varchar(128);not null" json:"display_name"`
	Config           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	RollbackFromUUID *string        `gorm:"type:uuid" json:"rollback_from_uuid,omitempty"`
	PublishedAt      *time.Time     `json:"published_at,omitempty"`
	PublishedBy      string         `gorm:"type:varchar(128)" json:"published_by,omitempty"`
	CreatedBy        string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
}

func (LocalRAGProfileVersion) TableName() string { return S(TableLocalKnowledgeRAGProfiles) }
func (m *LocalRAGProfileVersion) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalKnowledgeIngestionJob struct {
	BaseNoTenantModel
	UUID                string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID          string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID           string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	SourceID            string         `gorm:"type:varchar(128);not null" json:"source_id"`
	SourceType          string         `gorm:"type:varchar(32);not null" json:"source_type"`
	Status              string         `gorm:"type:varchar(32);not null;default:pending;index" json:"status"`
	Priority            string         `gorm:"type:varchar(16);not null;default:normal" json:"priority"`
	RetryCount          int            `gorm:"not null;default:0" json:"retry_count"`
	ChunkTotal          int            `gorm:"not null;default:0" json:"chunk_total"`
	ProgressPercent     int            `gorm:"not null;default:0" json:"progress_percent"`
	SummaryChunkCount   int            `gorm:"not null;default:0" json:"summary_chunk_count"`
	ParagraphChunkCount int            `gorm:"not null;default:0" json:"paragraph_chunk_count"`
	ChunkCoveredPct     float64        `gorm:"not null;default:0" json:"chunk_covered_pct"`
	EmbeddingSuccessPct float64        `gorm:"not null;default:0" json:"embedding_success_pct"`
	MaskingCoveragePct  float64        `gorm:"not null;default:0" json:"masking_coverage_pct"`
	ArtifactBundleUUID  *string        `gorm:"type:uuid;index" json:"artifact_bundle_uuid,omitempty"`
	ErrorCode           string         `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	BlockedReason       string         `gorm:"type:text" json:"blocked_reason,omitempty"`
	SubmittedBy         string         `gorm:"type:varchar(128)" json:"submitted_by,omitempty"`
	StartedAt           *time.Time     `json:"started_at,omitempty"`
	CompletedAt         *time.Time     `json:"completed_at,omitempty"`
	MetricsSnapshot     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metrics_snapshot"`
}

func (LocalKnowledgeIngestionJob) TableName() string { return S(TableLocalKnowledgeIngestionJobs) }
func (m *LocalKnowledgeIngestionJob) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalKnowledgeIndexJob struct {
	BaseNoTenantModel
	UUID         string     `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID   string     `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID    string     `gorm:"type:uuid;not null;index" json:"space_uuid"`
	DocumentUUID *string    `gorm:"type:uuid;index" json:"document_uuid,omitempty"`
	Operation    string     `gorm:"type:varchar(32);not null;index" json:"operation"`
	Status       string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode    string     `gorm:"type:varchar(64)" json:"error_code,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func (LocalKnowledgeIndexJob) TableName() string { return S(TableLocalKnowledgeIndexJobs) }
func (m *LocalKnowledgeIndexJob) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalKnowledgeVectorIndex struct {
	BaseNoTenantModel
	UUID                string     `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID          string     `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID           string     `gorm:"type:uuid;not null;uniqueIndex:uk_local_vector_space_key,priority:1;index" json:"space_uuid"`
	IndexKey            string     `gorm:"type:varchar(128);not null;uniqueIndex:uk_local_vector_space_key,priority:2;index" json:"index_key"`
	VectorTable         string     `gorm:"type:varchar(128);not null" json:"table_name"`
	Dimensions          int        `gorm:"not null" json:"dimensions"`
	EmbeddingProvider   string     `gorm:"type:varchar(64);not null" json:"embedding_provider"`
	EmbeddingModel      string     `gorm:"type:varchar(128);not null" json:"embedding_model"`
	EmbeddingProfileRef string     `gorm:"type:varchar(128)" json:"embedding_profile_ref,omitempty"`
	Status              string     `gorm:"type:varchar(32);not null;default:creating;index" json:"status"`
	LastUsedAt          *time.Time `json:"last_used_at,omitempty"`
	LastError           string     `gorm:"type:text" json:"last_error,omitempty"`
}

func (LocalKnowledgeVectorIndex) TableName() string { return S(TableLocalKnowledgeVectorIndexes) }
func (m *LocalKnowledgeVectorIndex) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
