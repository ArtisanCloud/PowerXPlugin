package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PluginSystemConfig stores tenant-scoped settings that are not part of an
// individual business profile. References use tenant UUID only.
type PluginSystemConfig struct {
	BaseNoTenantModel
	UUID       string         `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_plugin_system_configs_uuid" json:"uuid"`
	TenantUUID string         `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_plugin_system_configs_tenant_key,priority:1;index" json:"tenant_uuid"`
	Key        string         `gorm:"column:key;type:varchar(128);not null;uniqueIndex:uk_plugin_system_configs_tenant_key,priority:2" json:"key"`
	Value      datatypes.JSON `gorm:"column:value;type:jsonb;not null" json:"value"`
}

func (PluginSystemConfig) TableName() string { return S(TablePluginSystemConfigs) }
func (m *PluginSystemConfig) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
