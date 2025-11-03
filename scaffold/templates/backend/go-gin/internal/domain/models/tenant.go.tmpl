// internal/domain/models/plugin/tenant_ext.go
package models

import (
	"time"

	"gorm.io/datatypes"
)

type PluginTenantExt struct {
	BaseNoTenantModel                // 含 tenant_id/软删/时间
	Status            int16          `gorm:"type:smallint;not null;default:0" json:"status"`
	Plan              string         `gorm:"type:varchar(32);not null;default:'free'" json:"plan"`
	Flags             datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"flags"`
	Config            datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Region            string         `gorm:"type:varchar(32)" json:"region"`
	ExpireAt          *time.Time     `gorm:"type:timestamptz" json:"expire_at"`
	LastSyncAt        *time.Time     `gorm:"type:timestamptz" json:"last_sync_at"`
	LastError         string         `gorm:"type:text" json:"last_error"`
}

func (PluginTenantExt) TableName() string { return S(TablePluginTenantExt) }
