package migrate

import (
	"context"
	"fmt"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Record "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/regression"
	"gorm.io/gorm"
)

func MigrateRegression(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if err := db.WithContext(ctx).AutoMigrate(&Record.RunRecord{}, &Record.CaseExecutionRecord{}, &Record.EventRecord{}); err != nil {
		return err
	}
	indexes := []struct{ name, table, columns string }{{"regression_request_unique", Models.TableRegressionRuns, "tenant_uuid, actor_uuid, client_request_uuid"}, {"regression_case_order_unique", Models.TableRegressionCaseExecutions, "tenant_uuid, run_uuid, ordinal"}, {"regression_event_order_unique", Models.TableRegressionEvents, "tenant_uuid, run_uuid, sequence"}}
	for _, i := range indexes {
		if err := db.WithContext(ctx).Exec(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)", i.name, Models.S(i.table), i.columns)).Error; err != nil {
			return err
		}
	}
	if db.Dialector.Name() == "postgres" {
		for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents} {
			name := Models.S(table)
			for _, sql := range []string{"ALTER TABLE " + name + " ENABLE ROW LEVEL SECURITY", "ALTER TABLE " + name + " FORCE ROW LEVEL SECURITY", "DROP POLICY IF EXISTS regression_tenant ON " + name, "CREATE POLICY regression_tenant ON " + name + " USING (tenant_uuid = current_setting('app.tenant_uuid', true)::uuid) WITH CHECK (tenant_uuid = current_setting('app.tenant_uuid', true)::uuid)"} {
				if err := db.WithContext(ctx).Exec(sql).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func RollbackRegression(ctx context.Context, db *gorm.DB) error {
	for _, table := range []string{Models.TableRegressionEvents, Models.TableRegressionCaseExecutions, Models.TableRegressionRuns} {
		if err := db.WithContext(ctx).Migrator().DropTable(Models.S(table)); err != nil {
			return err
		}
	}
	return nil
}
