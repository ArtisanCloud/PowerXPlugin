package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LocalKnowledgeJobChunk preserves the content produced by one ingestion job.
// The active index can replace LocalKnowledgeChunk rows without rewriting history.
type LocalKnowledgeJobChunk struct {
	BaseNoTenantModel
	UUID         string         `gorm:"type:uuid;not null;uniqueIndex" json:"uuid"`
	TenantUUID   string         `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	SpaceUUID    string         `gorm:"type:uuid;not null;index" json:"space_uuid"`
	DocumentUUID string         `gorm:"type:uuid;not null;index" json:"document_uuid"`
	JobUUID      string         `gorm:"type:uuid;not null;uniqueIndex:uk_local_knowledge_job_chunk_ordinal,priority:1;index" json:"job_uuid"`
	Ordinal      int            `gorm:"not null;uniqueIndex:uk_local_knowledge_job_chunk_ordinal,priority:2" json:"ordinal"`
	Kind         string         `gorm:"type:text;not null" json:"kind"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	Metadata     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
}

func (LocalKnowledgeJobChunk) TableName() string { return S(TableLocalKnowledgeJobChunks) }

func (m *LocalKnowledgeJobChunk) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
