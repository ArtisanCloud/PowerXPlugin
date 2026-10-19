package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// The caller supplies a disposable PostgreSQL instance, never the normal
// application DSN. These checks run with PostgreSQL enforcing read-only SQL.
func TestRecoverySchemaPostgres(t *testing.T) {
	dsn := os.Getenv("POWERX_TEST_RECOVERY_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL required")
	}
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	defer func() { require.NoError(t, Close()) }()
	require.NoError(t, db.Exec("CREATE SCHEMA recovery_schema_fixture").Error)
	require.NoError(t, db.Exec("SET default_transaction_read_only = on").Error)
	require.NoError(t, prepareSchema("recovery_schema_fixture"))
	require.True(t, IsReadOnly())
	require.ErrorContains(t, prepareSchema("recovery_missing_schema"), "RECOVERY_SCHEMA_MISSING")
	// Read-only mode must never create the missing schema.
	var exists bool
	require.NoError(t, db.Raw("SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = ?)", "recovery_missing_schema").Scan(&exists).Error)
	require.False(t, exists)
	require.NoError(t, setDefaultSchema("recovery_schema_fixture"))
	require.Error(t, db.Exec("CREATE TABLE recovery_schema_fixture.forbidden(id int)").Error)
	require.NoError(t, db.Exec("SET default_transaction_read_only = off").Error)
	require.NoError(t, prepareSchema("recovery_writable_schema"))
	require.False(t, IsReadOnly())
	require.NoError(t, db.Raw("SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = ?)", "recovery_writable_schema").Scan(&exists).Error)
	require.True(t, exists)
}

func TestRecoveryBackgroundWorkersPolicy(t *testing.T) {
	previous := readOnlyDatabase
	t.Cleanup(func() { readOnlyDatabase = previous })
	readOnlyDatabase = false
	t.Setenv("POWERX_DISABLE_BACKGROUND_WORKERS", "false")
	require.False(t, BackgroundWorkersDisabled())
	t.Setenv("POWERX_DISABLE_BACKGROUND_WORKERS", "true")
	require.True(t, BackgroundWorkersDisabled())
	require.False(t, IsReadOnly())
	readOnlyDatabase = true
	t.Setenv("POWERX_DISABLE_BACKGROUND_WORKERS", "false")
	require.True(t, BackgroundWorkersDisabled())
}
