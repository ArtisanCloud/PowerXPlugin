package regression

import (
	"context"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Config "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestRegressionProfilesDoNotSwitchGlobalConfiguration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	tenant := uuid.NewString()
	cfg := &Config.Config{Database: &Config.DatabaseConfig{DSN: "file:fixture"}, Context: &Config.ContextConfig{ProviderMode: "local"}}
	deps := &App.Deps{DB: db, Config: cfg}
	before := string(Regression.JSON(cfg))
	r := Regression.NewRegistry()
	p := Config.RegressionProfileConfig{UUID: uuid.NewString(), Key: "live", Kind: "live", TitleKey: "regression.profiles.live", FixtureNamespace: "regression", TenantUUID: tenant, MetadataResourceType: "fixture.resource", MetadataResourceUUID: uuid.NewString()}
	if err = registerProfile(r, deps, p); err != nil {
		t.Fatal(err)
	}
	_, factory, err := r.Profile(p.UUID)
	if err != nil {
		t.Fatal(err)
	}
	a := Regression.Actor{TenantUUID: tenant, UUID: uuid.NewString(), PluginID: "com.powerx.fixture", RequestID: uuid.NewString(), TraceID: uuid.NewString()}
	_, evidence, err := factory(context.Background(), a)
	if Regression.CodeOf(err) != Regression.DependencyBlocked || evidence[0].ActualMode != "real" {
		t.Fatal(evidence, err)
	}
	if string(Regression.JSON(cfg)) != before {
		t.Fatal(Regression.InvalidRequest)
	}
}
