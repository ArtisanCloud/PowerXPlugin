package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LocalAISetting is a tenant-scoped AI profile. Source partitions local and
// PowerX-backed profiles for the same environment and modality. Secret
// material never enters this table: CredentialRef is an opaque reference
// managed outside this model.
type LocalAISetting struct {
	BaseNoTenantModel
	UUID          string         `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_local_ai_settings_uuid" json:"uuid"`
	TenantUUID    string         `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_local_ai_settings_tenant_env_modality_source,priority:1;index" json:"tenant_uuid"`
	Environment   string         `gorm:"column:environment;type:varchar(32);not null;uniqueIndex:uk_local_ai_settings_tenant_env_modality_source,priority:2" json:"environment"`
	Modality      string         `gorm:"column:modality;type:varchar(32);not null;uniqueIndex:uk_local_ai_settings_tenant_env_modality_source,priority:3" json:"modality"`
	Source        string         `gorm:"column:source;type:varchar(16);not null;default:local;uniqueIndex:uk_local_ai_settings_tenant_env_modality_source,priority:4" json:"source"`
	Provider      string         `gorm:"column:provider;type:varchar(64);not null" json:"provider"`
	App           string         `gorm:"column:app;type:varchar(64)" json:"app,omitempty"`
	ModelKey      string         `gorm:"column:model_key;type:varchar(128);not null" json:"model_key"`
	Endpoint      string         `gorm:"column:endpoint;type:text" json:"endpoint,omitempty"`
	CredentialRef string         `gorm:"column:credential_ref;type:text" json:"credential_ref,omitempty"`
	Parameters    datatypes.JSON `gorm:"column:parameters;type:jsonb;not null" json:"parameters"`
}

func (LocalAISetting) TableName() string { return S(TableLocalAISettings) }

func (m *LocalAISetting) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
