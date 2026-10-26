package migrate

import (
	"context"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Record "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/regression"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestRegressionMigrationIdempotentAndRollback(t *testing.T) {
	Models.ForceSchemaForTests("")
	t.Cleanup(func() { Models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	ctx := context.Background()
	if err = db.Exec("CREATE TABLE regression_migration_sentinel (uuid TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("INSERT INTO regression_migration_sentinel (uuid) VALUES (?)", "55555555-5555-4555-8555-555555555555").Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = MigrateRegression(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents} {
		if !db.Migrator().HasTable(table) {
			t.Fatal(table)
		}
	}
	bad := Record.RunRecord{}
	if err = db.Create(&bad).Error; err == nil {
		t.Fatal(gorm.ErrInvalidData)
	}
	if err = RollbackRegression(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents} {
		if db.Migrator().HasTable(table) {
			t.Fatal(table)
		}
	}
	var remaining int64
	if err = db.Table("regression_migration_sentinel").Count(&remaining).Error; err != nil || remaining != 1 {
		t.Fatalf("regression_migration_sentinel count=%d error=%v", remaining, err)
	}
}
