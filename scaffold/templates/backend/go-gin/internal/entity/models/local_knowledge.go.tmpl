package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LocalKnowledgeSpace struct {
	BaseNoTenantModel
	UUID           string `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID     string `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_space_space_name,priority:1;index" json:"tenant_uuid"`
	SpaceName      string `gorm:"column:space_name;type:varchar(128);not null;uniqueIndex:uk_local_knowledge_space_space_name,priority:2" json:"name"`
	DepartmentCode string `gorm:"type:varchar(64);not null;default:''" json:"department_code"`
	Status         string `gorm:"type:varchar(32);not null;default:active;index" json:"status"`
	QuotaCPU       int    `gorm:"not null;default:2" json:"quota_cpu"`
	QuotaStorageGB int    `gorm:"not null;default:50" json:"quota_storage_gb"`
	// PolicyTemplateVersionUUID is optional. A pointer is required here so an
	// unset relationship is persisted as SQL NULL rather than the invalid UUID
	// literal "" on PostgreSQL.
	PolicyTemplateVersionUUID *string        `gorm:"type:uuid" json:"policy_template_version_uuid,omitempty"`
	IngestionProfileKey       string         `gorm:"type:varchar(128);not null;default:default;index" json:"ingestion_profile_key"`
	IndexProfileKey           string         `gorm:"type:varchar(128);not null;default:default;index" json:"index_profile_key"`
	RAGProfileKey             string         `gorm:"type:varchar(128);not null;default:default;index" json:"rag_profile_key"`
	EmbeddingProfileKey       string         `gorm:"type:varchar(128);not null;default:'';index" json:"embedding_profile_key"`
	ActiveVectorIndexKey      string         `gorm:"type:varchar(128);not null;default:''" json:"active_vector_index_key,omitempty"`
	FeatureFlags              datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"feature_flags"`
	RetireAt                  *time.Time     `json:"retire_at,omitempty"`
	RetentionExpiresAt        *time.Time     `json:"retention_expires_at,omitempty"`
	CreatedBy                 string         `gorm:"type:varchar(128)" json:"created_by,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(128)" json:"updated_by,omitempty"`
	LastAuditedAt             *time.Time     `json:"last_audited_at,omitempty"`
	AuditToken                string         `gorm:"type:varchar(128)" json:"audit_token,omitempty"`
}

func (LocalKnowledgeSpace) TableName() string { return S(TableLocalKnowledgeSpaces) }
func (m *LocalKnowledgeSpace) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalKnowledgeDocument struct {
	BaseNoTenantModel
	UUID               string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID         string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_document_uri,priority:1;index" json:"tenant_uuid"`
	SpaceUUID          string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_document_uri,priority:2;index" json:"space_uuid"`
	Title              string         `gorm:"type:varchar(255);not null" json:"title"`
	URI                string         `gorm:"type:text;not null;default:'';uniqueIndex:uk_local_knowledge_document_uri,priority:3" json:"uri"`
	Content            string         `gorm:"type:text;not null" json:"content"`
	ContentType        string         `gorm:"type:varchar(128);not null;default:text/markdown" json:"content_type"`
	Checksum           string         `gorm:"type:char(64);not null;default:'';index" json:"checksum"`
	Version            string         `gorm:"type:varchar(128);not null;default:1" json:"version"`
	Tags               datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tags"`
	SourceType         string         `gorm:"type:varchar(24);not null;default:manual" json:"source_type"`
	EffectiveFrom      *time.Time     `gorm:"index" json:"effective_from,omitempty"`
	EffectiveTo        *time.Time     `gorm:"index" json:"effective_to,omitempty"`
	AccessScope        string         `gorm:"type:varchar(32);not null;default:space_department;index" json:"access_scope"`
	AllowedMemberUUIDs datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"allowed_member_uuids"`
	Status             string         `gorm:"column:index_status;type:varchar(32);not null;default:queued;index" json:"index_status"`
	IndexedAt          *time.Time     `json:"indexed_at,omitempty"`
	ChunkCount         int            `gorm:"not null;default:0" json:"chunk_count"`
}

// LocalKnowledgeRoute is the explicit A1 domain router map. Both endpoints
// are plugin-owned knowledge space UUIDs and are always tenant-scoped.
type LocalKnowledgeRoute struct {
	BaseNoTenantModel
	UUID            string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID      string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	RouterSpaceUUID string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_route,priority:1;index" json:"router_space_uuid"`
	TargetSpaceUUID string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_route,priority:2;index" json:"target_space_uuid"`
	Keywords        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"keywords"`
	Priority        int            `gorm:"not null;default:100;index" json:"priority"`
	Enabled         bool           `gorm:"not null;default:true;index" json:"enabled"`
}

// LocalKnowledgeKGNode mirrors Core's knowledge_kg_nodes storage while adding
// a stable UUID for cross-service references. NodeID remains the graph's
// space-local identity; Props carries labels, aliases and chunk provenance.
type LocalKnowledgeKGNode struct {
	BaseNoTenantModel
	UUID         string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID   string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID    string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_kg_node,priority:1;index" json:"space_uuid"`
	DocumentUUID string         `gorm:"type:uuid;not null;index" json:"document_uuid"`
	NodeID       string         `gorm:"type:varchar(255);not null;uniqueIndex:uk_local_knowledge_kg_node,priority:2;index" json:"node_id"`
	NodeType     string         `gorm:"type:varchar(64);not null;default:entity;index" json:"node_type"`
	Props        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"props"`
}

func (LocalKnowledgeKGNode) TableName() string { return S(TableLocalKnowledgeKGNodes) }
func (m *LocalKnowledgeKGNode) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

// LocalKnowledgeKGEdge mirrors Core's knowledge_kg_edges storage. Both graph
// endpoints are node UUIDs, never an implicit numeric database id.
type LocalKnowledgeKGEdge struct {
	BaseNoTenantModel
	UUID         string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID   string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID    string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_kg_edge,priority:1;index" json:"space_uuid"`
	DocumentUUID string         `gorm:"type:uuid;not null;index" json:"document_uuid"`
	EdgeID       string         `gorm:"type:varchar(255);not null;uniqueIndex:uk_local_knowledge_kg_edge,priority:2;index" json:"edge_id"`
	SrcNodeUUID  string         `gorm:"type:uuid;not null;index" json:"src_node_uuid"`
	DstNodeUUID  string         `gorm:"type:uuid;not null;index" json:"dst_node_uuid"`
	Predicate    string         `gorm:"type:varchar(128);not null;index" json:"predicate"`
	Props        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"props"`
}

func (LocalKnowledgeKGEdge) TableName() string { return S(TableLocalKnowledgeKGEdges) }
func (m *LocalKnowledgeKGEdge) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

func (LocalKnowledgeRoute) TableName() string { return S(TableLocalKnowledgeRoutes) }
func (m *LocalKnowledgeRoute) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

func (LocalKnowledgeDocument) TableName() string { return S(TableLocalKnowledgeDocuments) }
func (m *LocalKnowledgeDocument) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type LocalKnowledgeChunk struct {
	BaseNoTenantModel
	UUID         string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_chunk_space_chunk,priority:2" json:"uuid"`
	TenantUUID   string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID    string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_chunk_space_chunk,priority:1;index" json:"space_uuid"`
	DocumentUUID string         `gorm:"type:uuid;not null;index" json:"document_uuid"`
	JobUUID      *string        `gorm:"type:uuid;index" json:"job_uuid,omitempty"`
	Ordinal      int            `gorm:"not null" json:"ordinal"`
	Kind         string         `gorm:"type:text;not null;default:chunk" json:"kind"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	Metadata     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
}

func (LocalKnowledgeChunk) TableName() string { return S(TableLocalKnowledgeChunks) }
func (m *LocalKnowledgeChunk) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
