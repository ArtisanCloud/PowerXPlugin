package ai_settings

import (
	"context"
	"errors"
	"time"

	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func onlineConflict() error {
	return &powerxai.HTTPError{StatusCode: 409, ErrorCode: "AI_MODEL_SYNC_STALE", ReasonCode: "AI_MODEL_SYNC_STALE"}
}

func (r *LocalSettingsRepository) withOnlineTenantTx(ctx context.Context, tenant string, fn func(*gorm.DB) error) error {
	if _, err := uuid.Parse(tenant); err != nil {
		return &powerxai.HTTPError{StatusCode: 400, ReasonCode: "AI_REQUEST_INVALID"}
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// set_config supports bound parameters with PostgreSQL's extended
		// query protocol and keeps the setting local to this transaction.
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT set_config('app.tenant_uuid', ?, true)", tenant).Error; err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

func (r *LocalSettingsRepository) OnlineCatalog(ctx context.Context, tenant, env, provider, app string) (*models.LocalAIOnlineCatalog, error) {
	var present bool
	var check error
	switch r.DB.Dialector.Name() {
	case "postgres":
		check = r.DB.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL", (models.LocalAIOnlineCatalog{}).TableName()).Scan(&present).Error
	case "sqlite":
		check = r.DB.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", (models.LocalAIOnlineCatalog{}).TableName()).Scan(&present).Error
	default:
		return nil, &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_LOCAL_CATALOG_STORE_UNSUPPORTED"}
	}
	if check != nil {
		return nil, check
	}
	if !present {
		return nil, &powerxai.HTTPError{StatusCode: 503, ErrorCode: "AI_LOCAL_CATALOG_MIGRATION_REQUIRED", ReasonCode: "AI_LOCAL_CATALOG_MIGRATION_REQUIRED"}
	}
	var result models.LocalAIOnlineCatalog
	err := r.withOnlineTenantTx(ctx, tenant, func(tx *gorm.DB) error {
		return tx.Where("tenant_uuid = ? AND environment = ? AND provider = ? AND app = ?", tenant, env, provider, app).First(&result).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplaceOnlineCatalog compares revisions so a slow refresh cannot overwrite a
// newer refresh or an administrator's explicit classification.
func (r *LocalSettingsRepository) ReplaceOnlineCatalog(ctx context.Context, row *models.LocalAIOnlineCatalog, expected string) (*models.LocalAIOnlineCatalog, error) {
	err := r.withOnlineTenantTx(ctx, row.TenantUUID, func(tx *gorm.DB) error {
		where := tx.Model(&models.LocalAIOnlineCatalog{}).Where("tenant_uuid = ? AND environment = ? AND provider = ? AND app = ?", row.TenantUUID, row.Environment, row.Provider, row.App)
		if expected == "" {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return onlineConflict()
			}
			return nil
		}
		result := where.Where("revision = ?", expected).Updates(map[string]any{"fingerprint": row.Fingerprint, "revision": row.Revision, "protocol": row.Protocol, "source": row.Source, "models": row.Models, "updated_at": time.Now().UTC()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return onlineConflict()
		}
		return tx.Where("tenant_uuid = ? AND environment = ? AND provider = ? AND app = ?", row.TenantUUID, row.Environment, row.Provider, row.App).First(row).Error
	})
	return row, err
}
