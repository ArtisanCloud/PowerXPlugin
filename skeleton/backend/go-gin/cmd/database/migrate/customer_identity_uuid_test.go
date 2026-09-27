package migrate

import (
	"context"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestCustomerIdentityUUIDBackfillIsStable(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:identity-migration?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE customer_auth_identities (id INTEGER PRIMARY KEY, deleted_at DATETIME)").Error)
	require.NoError(t, db.Exec("INSERT INTO customer_auth_identities(id) VALUES(1),(2)").Error)
	require.NoError(t, ensureCustomerIdentityUUIDs(context.Background(), db))
	var first []string
	require.NoError(t, db.Table("customer_auth_identities").Order("id").Pluck("identity_uuid", &first).Error)
	require.Len(t, first, 2)
	for _, id := range first {
		_, err = uuid.Parse(id)
		require.NoError(t, err)
	}
	require.NotEqual(t, first[0], first[1])
	require.NoError(t, ensureCustomerIdentityUUIDs(context.Background(), db))
	var second []string
	require.NoError(t, db.Table("customer_auth_identities").Order("id").Pluck("identity_uuid", &second).Error)
	require.Equal(t, first, second)
}
