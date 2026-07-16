package migrate

import (
	"context"
	"testing"

	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	identitymodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestMigratePluginModelsIncludesFederatedIAMTables(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() {
		EntityModels.ForceSchemaForTests("public")
	})

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}

	mustHave := []interface{}{
		&identitymodel.FederatedExternalIdentity{},
		&identitymodel.FederatedBinding{},
		&identitymodel.FederatedLoginChallenge{},
		&identitymodel.FederatedRiskEvent{},
	}
	for _, model := range mustHave {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("HasTable(%T) = false, want true", model)
		}
	}
}
