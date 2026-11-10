package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LocalAIOnlineCatalog contains metadata only; credentials and their fingerprints
// never leave the server. A row is owned by one tenant/environment/connection.
type LocalAIOnlineCatalog struct {
	BaseNoTenantModel
	UUID        string         `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_ai_online_uuid" json:"uuid"`
	TenantUUID  string         `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_ai_online_scope,priority:1" json:"tenant_uuid"`
	Environment string         `gorm:"column:environment;type:varchar(32);not null;uniqueIndex:uk_ai_online_scope,priority:2" json:"environment"`
	Provider    string         `gorm:"column:provider;type:varchar(64);not null;uniqueIndex:uk_ai_online_scope,priority:3" json:"provider"`
	App         string         `gorm:"column:app;type:varchar(128);not null;uniqueIndex:uk_ai_online_scope,priority:4" json:"app"`
	Fingerprint string         `gorm:"column:fingerprint;type:varchar(64);not null" json:"-"`
	Revision    string         `gorm:"column:revision;type:uuid;not null" json:"revision"`
	Protocol    string         `gorm:"column:protocol;type:varchar(32);not null" json:"protocol"`
	Source      string         `gorm:"column:source;type:text;not null" json:"source"`
	Models      datatypes.JSON `gorm:"column:models;type:jsonb;not null" json:"models"`
}

func (LocalAIOnlineCatalog) TableName() string { return S(TableLocalAIOnlineCatalogs) }
func (m *LocalAIOnlineCatalog) BeforeCreate(*gorm.DB) error {
	if m.UUID == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
