package migrate

import (
	"context"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	adminconsoleModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/admin_console"
	marketplaceModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/marketplace"
	operationsModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/operations"
	runtimeOpsModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/runtime_ops"
	securityModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/security"
	templateModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	toolgrantModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/tool_grant"
	"gorm.io/gorm"
)

var pluginTables = []interface{}{
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
}

// MigratePluginModels 只做 AutoMigrate（最小实现）
func MigratePluginModels(ctx context.Context, db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "sqlite", "sqlite3":
		return db.AutoMigrate(&templateModel.Template{})
	}
	return db.AutoMigrate(pluginTables...)
}

func ResetDatabase(ctx context.Context, db *gorm.DB, cfg *config.DatabaseConfig) error {
	if db.Dialector.Name() == "sqlite" || strings.TrimSpace(cfg.Schema) == "" {
		return db.WithContext(ctx).Migrator().DropTable(&templateModel.Template{})
	}

	// 如果你用 GORM，可以直接 drop 所有表
	// 或者先获取表名，再循环 drop
	// 这里举例简单版本：
	err := db.Exec("DROP SCHEMA " + cfg.Schema + " CASCADE; CREATE SCHEMA " + cfg.Schema + ";").Error
	if err != nil {
		return err
	}
	return nil
}
