package migrate

import (
	"context"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/config"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models"
	adminconsoleModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/admin_console"
	marketplaceModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/marketplace"
	operationsModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/operations"
	runtimeOpsModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/runtime_ops"
	securityModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/security"
	templateModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/template"
	toolgrantModel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/tool_grant"
	"gorm.io/gorm"
)

// MigratePluginModels 只做 AutoMigrate（最小实现）
func MigratePluginModels(ctx context.Context, db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PluginCredential{},
		&models.PluginTenantExt{},
		&templateModel.Template{},
		&marketplaceModel.Listing{},
		&marketplaceModel.ListingAsset{},
		&marketplaceModel.ListingVersion{},
		&marketplaceModel.ChecklistRun{},
		&marketplaceModel.ChecklistItem{},
		&marketplaceModel.PricingPlan{},
		&marketplaceModel.PlanTier{},
		&marketplaceModel.License{},
		&marketplaceModel.LicenseEvent{},
		&marketplaceModel.TaxTransaction{},
		&runtimeOpsModel.MCPSession{},
		&runtimeOpsModel.RuntimeAuditEvent{},
		&runtimeOpsModel.QuotaLedger{},
		&runtimeOpsModel.MarketplaceOverage{},
		&operationsModel.SupportChannel{},
		&operationsModel.SupportTicket{},
		&operationsModel.SupportTicketEvent{},
		&operationsModel.ReadinessChecklistItem{},
		&operationsModel.SLAProfile{},
		&operationsModel.SLAAdjustment{},
		&operationsModel.Incident{},
		&operationsModel.IncidentTimelineEntry{},
		&operationsModel.IncidentChecklistItem{},
		&securityModel.BaselineChecklist{},
		&securityModel.AuditReport{},
		&toolgrantModel.Revocation{},
		&toolgrantModel.UsageEvent{},
		&adminconsoleModel.AuditEvent{},
		&adminconsoleModel.ConfigChange{},
		&adminconsoleModel.JobRun{},
	)
}

func ResetDatabase(ctx context.Context, db *gorm.DB, cfg *config.DatabaseConfig) error {
	// 如果你用 GORM，可以直接 drop 所有表
	// 或者先获取表名，再循环 drop
	// 这里举例简单版本：
	err := db.Exec("DROP SCHEMA " + cfg.Schema + " CASCADE; CREATE SCHEMA " + cfg.Schema + ";").Error
	if err != nil {
		return err
	}
	return nil
}
