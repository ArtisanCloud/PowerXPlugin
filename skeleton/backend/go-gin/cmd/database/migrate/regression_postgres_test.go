package migrate

import (
	"context"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Config "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Template "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	Repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/regression"
	Service "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
	"time"
)

// The test requires an isolated server: it creates a role and a private schema.
func TestRegressionPostgresRLS(t *testing.T) {
	dsn := os.Getenv("REGRESSION_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("REGRESSION_POSTGRES_TEST_DSN")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	schema := "regression_" + suffix
	role := "regression_actor_" + suffix
	if err = db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	if err = db.Exec("CREATE ROLE " + role + " NOLOGIN NOSUPERUSER NOBYPASSRLS").Error; err != nil {
		t.Fatal(err)
	}
	defer func() { db.Exec("DROP OWNED BY " + role); db.Exec("DROP ROLE " + role) }()
	Models.ForceSchemaForTests(schema)
	defer Models.ForceSchemaForTests("public")
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err = MigrateRegression(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	if err = db.AutoMigrate(&Template.Template{}); err != nil {
		t.Fatal(err)
	}
	cfg := &Config.Config{DevMode: true, Logging: &Config.LoggingConfig{DebugMode: true}, Server: &Config.ServerConfig{Mode: "debug"}, Regression: Config.RegressionConfig{Enabled: true, Environment: "test", Profiles: []Config.RegressionProfileConfig{{UUID: "44444444-4444-4444-8444-444444444444", Key: "controlled", Kind: "controlled", TitleKey: "regression.profiles.controlled", FixtureNamespace: "regression"}}}}
	deps := &App.Deps{Config: cfg, DB: db}
	runner, err := Service.NewService(ctx, deps)
	if err != nil || runner == nil {
		t.Fatalf("REGRESSION_POSTGRES_STARTUP error=%v", err)
	}
	stop, cancel := context.WithTimeout(ctx, time.Second)
	if err = runner.Close(stop); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	if err = db.Exec("GRANT USAGE ON SCHEMA " + schema + " TO " + role).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA " + schema + " TO " + role).Error; err != nil {
		t.Fatal(err)
	}
	tenantA, tenantB := uuid.NewString(), uuid.NewString()
	runA, runB := uuid.NewString(), uuid.NewString()
	for _, entry := range []struct{ tenant, run string }{{tenantA, runA}, {tenantB, runB}} {
		if err = db.Exec("INSERT INTO "+Models.S(Models.TableRegressionRuns)+" (uuid,tenant_uuid,actor_uuid,profile_uuid,client_request_uuid,profile_snapshot,selection_snapshot,input_snapshot,source_snapshot,phase,verdict,cleanup_state,revision,created_at,updated_at,reason_code,message_key,request_hash) VALUES (?, ?, ?, ?, ?, '{}','[]','{}','{}','queued','pending','pending',1,now(),now(),'','','fixture')", entry.run, entry.tenant, uuid.NewString(), uuid.NewString(), uuid.NewString()).Error; err != nil {
			t.Fatal(err)
		}
	}
	store, err := Repository.NewRegressionRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Get(ctx, tenantA, runA); err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT set_config('app.tenant_uuid', ?, true)", tenantA).Error; err != nil {
			return err
		}
		var rows []struct{ UUID string }
		if err := tx.Table(Models.S(Models.TableRegressionRuns)).Select("uuid").Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) != 1 || rows[0].UUID != runA {
			t.Fatal(rows)
		}
		for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents} {
			var flags struct{ Enabled, Forced bool }
			if err := tx.Raw("SELECT relrowsecurity AS enabled, relforcerowsecurity AS forced FROM pg_class WHERE oid = ?::regclass", Models.S(table)).Scan(&flags).Error; err != nil {
				return err
			}
			if !flags.Enabled || !flags.Forced {
				t.Fatal(table)
			}
		}
		result := tx.Table(Models.S(Models.TableRegressionRuns)).Where("uuid = ?", runB).Update("phase", "completed")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 0 {
			t.Fatal(result.RowsAffected)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT set_config('app.tenant_uuid', ?, true)", tenantA).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO "+Models.S(Models.TableRegressionRuns)+" (uuid,tenant_uuid,actor_uuid,profile_uuid,client_request_uuid,profile_snapshot,selection_snapshot,input_snapshot,source_snapshot,phase,verdict,cleanup_state,revision,created_at,updated_at,reason_code,message_key,request_hash) VALUES (?, ?, ?, ?, ?, '{}','[]','{}','{}','queued','pending','pending',1,now(),now(),'','','fixture')", uuid.NewString(), tenantB, uuid.NewString(), uuid.NewString(), uuid.NewString()).Error
	})
	if err == nil {
		t.Fatal("REGRESSION_TENANT_SCOPE")
	}
	if err = RollbackRegression(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err = Service.NewService(ctx, deps); Regression.CodeOf(err) != Regression.StoreUnavailable {
		t.Fatalf("REGRESSION_MISSING_TABLE error=%v", err)
	}
	for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents} {
		if db.Migrator().HasTable(Models.S(table)) {
			t.Fatal(table)
		}
	}
}
