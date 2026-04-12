package migrate

import (
	"context"
	"testing"

	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigratePluginModelsIncludesFederatedIAMTables(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() {
		EntityModels.ForceSchemaForTests("public")
	})

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}

	mustHave := []interface{}{
		&iammodel.FederatedExternalIdentity{},
		&iammodel.FederatedBinding{},
		&iammodel.FederatedLoginChallenge{},
		&iammodel.FederatedRiskEvent{},
	}
	for _, model := range mustHave {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("HasTable(%T) = false, want true", model)
		}
	}
}
