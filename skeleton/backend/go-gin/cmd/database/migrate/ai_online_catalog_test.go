package migrate

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIOnlineCatalogMigrationAndRollbackAreScopedAndIdempotent(t *testing.T) {
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	defer sql.Close()
	require.NoError(t, db.AutoMigrate(&models.LocalAISetting{}))
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		require.NoError(t, MigrateAIOnlineCatalog(ctx, db))
	}
	row := models.LocalAIOnlineCatalog{TenantUUID: uuid.NewString(), Environment: "development", Provider: "ollama", Revision: uuid.NewString(), Fingerprint: "fixture", Models: datatypes.JSON(`[]`)}
	require.NoError(t, db.Create(&row).Error)
	require.NotEmpty(t, row.UUID)
	for i := 0; i < 2; i++ {
		require.NoError(t, RollbackAIOnlineCatalog(ctx, db))
	}
	require.False(t, db.Migrator().HasTable(&models.LocalAIOnlineCatalog{}))
	require.True(t, db.Migrator().HasTable(&models.LocalAISetting{}))
}

// The caller must supply an isolated PostgreSQL server; this creates a private
// schema and an unprivileged role and removes only those objects afterwards.
func TestAIOnlineCatalogPostgresIsolationRevisionAndRLS(t *testing.T) {
	dsn := os.Getenv("AI_ONLINE_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("AI_ONLINE_POSTGRES_TEST_DSN")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	defer sql.Close()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	schema, role := "ai_online_"+suffix, "ai_online_actor_"+suffix
	require.NoError(t, db.Exec("CREATE SCHEMA "+schema).Error)
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN NOSUPERUSER NOBYPASSRLS").Error)
	defer func() { db.Exec("DROP OWNED BY " + role); db.Exec("DROP ROLE " + role) }()
	models.ForceSchemaForTests(schema)
	defer models.ForceSchemaForTests("public")
	ctx := context.Background()
	require.NoError(t, db.AutoMigrate(&models.LocalAISetting{}))
	require.NoError(t, db.Exec("ALTER TABLE "+(models.LocalAISetting{}).TableName()+" ALTER COLUMN model_key TYPE varchar(128)").Error)
	for i := 0; i < 2; i++ {
		require.NoError(t, MigrateAIOnlineCatalog(ctx, db))
	}
	var length int
	require.NoError(t, db.Raw("SELECT character_maximum_length FROM information_schema.columns WHERE table_schema=? AND table_name=? AND column_name='model_key'", schema, models.TableLocalAISettings).Scan(&length).Error)
	require.Equal(t, 512, length)
	tenantA, tenantB := uuid.NewString(), uuid.NewString()
	repo, err := repository.NewLocalSettingsRepository(db)
	require.NoError(t, err)
	for _, tenant := range []string{tenantA, tenantB} {
		_, err = repo.ReplaceOnlineCatalog(ctx, &models.LocalAIOnlineCatalog{TenantUUID: tenant, Environment: "development", Provider: "ollama", Revision: uuid.NewString(), Fingerprint: "fixture", Models: datatypes.JSON(`[]`)}, "")
		require.NoError(t, err)
	}
	row, err := repo.OnlineCatalog(ctx, tenantA, "development", "ollama", "")
	require.NoError(t, err)
	require.Equal(t, tenantA, row.TenantUUID)
	oldRevision := row.Revision
	row.Revision = uuid.NewString()
	_, err = repo.ReplaceOnlineCatalog(ctx, row, oldRevision)
	require.NoError(t, err)
	_, err = repo.ReplaceOnlineCatalog(ctx, row, oldRevision)
	require.Error(t, err)
	require.NoError(t, db.Exec("GRANT USAGE ON SCHEMA "+schema+" TO "+role).Error)
	require.NoError(t, db.Exec("GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA "+schema+" TO "+role).Error)
	require.NoError(t, db.Exec("GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA "+schema+" TO "+role).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT set_config('app.tenant_uuid',?,true)", tenantA).Error; err != nil {
			return err
		}
		var rows []models.LocalAIOnlineCatalog
		if err := tx.Find(&rows).Error; err != nil {
			return err
		}
		require.Len(t, rows, 1)
		require.Equal(t, tenantA, rows[0].TenantUUID)
		result := tx.Model(&models.LocalAIOnlineCatalog{}).Where("tenant_uuid = ?", tenantB).Update("protocol", "fixture.denied")
		require.NoError(t, result.Error)
		require.Zero(t, result.RowsAffected)
		return tx.Create(&models.LocalAIOnlineCatalog{TenantUUID: tenantA, Environment: "production", Provider: "ollama", Revision: uuid.NewString(), Fingerprint: "fixture", Models: datatypes.JSON(`[]`)}).Error
	}))
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + role).Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT set_config('app.tenant_uuid',?,true)", tenantA).Error; err != nil {
			return err
		}
		return tx.Create(&models.LocalAIOnlineCatalog{TenantUUID: tenantB, Environment: "production", Provider: "ollama", Revision: uuid.NewString(), Fingerprint: "fixture", Models: datatypes.JSON(`[]`)}).Error
	})
	require.Error(t, err)
	var flags struct{ Enabled, Forced bool }
	require.NoError(t, db.Raw("SELECT relrowsecurity AS enabled,relforcerowsecurity AS forced FROM pg_class WHERE oid=?::regclass", models.LocalAIOnlineCatalog{}.TableName()).Scan(&flags).Error)
	require.True(t, flags.Enabled)
	require.True(t, flags.Forced)
	require.NoError(t, RollbackAIOnlineCatalog(ctx, db))
}
