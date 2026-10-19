package regression

import (
	"context"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/regression"
	Scenarios "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression/scenarios"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"os"
	"strings"
)

func NewService(ctx context.Context, deps *App.Deps) (*Regression.Runner, error) {
	if deps == nil || deps.Config == nil || !deps.Config.Regression.Enabled {
		return nil, nil
	}
	cfg := deps.Config.Regression
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if deps.Config.IsProduction() || deps.DB == nil {
		return nil, Regression.NewError(Regression.StoreUnavailable)
	}
	for _, table := range []string{Models.TableRegressionRuns, Models.TableRegressionCaseExecutions, Models.TableRegressionEvents, Models.TableTemplate} {
		var exists bool
		if deps.DB.Dialector.Name() == "postgres" {
			if err := deps.DB.Raw("SELECT to_regclass(?) IS NOT NULL", Models.S(table)).Scan(&exists).Error; err != nil {
				return nil, Regression.NewError(Regression.StoreUnavailable)
			}
		} else {
			exists = deps.DB.Migrator().HasTable(Models.S(table))
		}
		if !exists {
			return nil, Regression.NewError(Regression.StoreUnavailable)
		}
	}
	store, err := Repository.NewRegressionRepository(deps.DB)
	if err != nil {
		return nil, err
	}
	registry := Regression.NewRegistry()
	if err = registry.RegisterSuite(Regression.SuiteDefinition{UUID: Scenarios.SuiteUUID, Key: "business", TitleKey: "regression.suites.business", DescriptionKey: "regression.suites.description"}); err != nil {
		return nil, err
	}
	for _, factory := range Scenarios.TemplateScenarios() {
		if err = registry.RegisterScenario(factory); err != nil {
			return nil, err
		}
	}
	if err = registry.RegisterScenario(func() Regression.Scenario { return &Scenarios.MetadataTagBindings{} }); err != nil {
		return nil, err
	}
	for _, p := range cfg.Profiles {
		if err = registerProfile(registry, deps, p); err != nil {
			return nil, err
		}
	}
	return Regression.NewRunner(ctx, registry, store, Regression.JSON(map[string]any{"plugin_version": App.PluginVersion, "source_revision": strings.TrimSpace(os.Getenv("POWERX_SOURCE_REVISION")), "source_version_available": strings.TrimSpace(os.Getenv("POWERX_SOURCE_REVISION")) != ""}), func(ctx context.Context, e Regression.RegressionEvent, err error) {
		log := deps.RuntimeLogger(ctx, "regression", map[string]any{"run_uuid": e.RunUUID, "tenant_uuid": e.TenantUUID, "trace_id": e.Meta.TraceID, "request_id": e.Meta.RequestID, "event_kind": e.Kind})
		if err != nil {
			log.Error(Regression.StoreUnavailable)
		} else {
			log.Debug(e.Kind)
		}
	})
}
