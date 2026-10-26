package regression

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Migrate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/cmd/database/migrate"
	Config "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Template "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	Middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	Service "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression"
	Scenarios "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression/scenarios"
	Templates "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/templates"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Exercise the registered business catalog, not synthetic scenarios: every case
// runs both in a batch and alone, including expected HTTP/partial failures.
func TestRegressionTemplateCatalogBatchAndIndependentRuns(t *testing.T) {
	Models.ForceSchemaForTests("")
	t.Cleanup(func() { Models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, _ := db.DB()
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { connection.Close() })
	ctx := context.Background()
	if err = Migrate.MigrateRegression(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&Template.Template{}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../../../../web-admin/nuxt/i18n/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var language struct {
		Regression struct{ Fixtures map[string]string }
	}
	if err = json.Unmarshal(b, &language); err != nil {
		t.Fatal(err)
	}
	values := language.Regression.Fixtures
	profile := uuid.NewString()
	cfg := &Config.Config{DevMode: true, Regression: Config.RegressionConfig{Enabled: true, Environment: "test", Profiles: []Config.RegressionProfileConfig{{UUID: profile, Key: "controlled", Kind: "controlled", TitleKey: "regression.profiles.controlled", FixtureNamespace: "regression"}}}}
	deps := &App.Deps{Config: cfg, DB: db}
	actor := Regression.Actor{TenantUUID: uuid.NewString(), UUID: uuid.NewString(), PluginID: App.PluginID, RequestID: uuid.NewString(), TraceID: uuid.NewString()}
	runner, err := Service.NewService(ctx, deps)
	if err != nil {
		t.Fatal(err)
	}
	definitions := []Regression.CaseDefinition{}
	inputs := map[string]any{}
	for _, definition := range runner.Registry.Cases() {
		if !Regression.Equal(definition.SupportedProfileKinds, []string{"controlled"}) {
			continue
		}
		definitions = append(definitions, definition)
		var schema struct{ Properties map[string]any }
		if err = json.Unmarshal(definition.InputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		available := map[string]string{"name": values["name"], "description": values["description"], "content": values["content"], "updated_content": values["updatedContent"]}
		if definition.UUID == Scenarios.TemplateValidationUUID {
			available["content"] = values["validationContent"]
		}
		fields := map[string]string{}
		for key := range schema.Properties {
			value, ok := available[key]
			if !ok || value == "" {
				t.Fatal(definition.Key, key)
			}
			fields[key] = value
		}
		inputs[definition.UUID] = fields
	}
	if len(definitions) != 8 || len(runner.Registry.Cases()) != 9 {
		t.Fatal(len(definitions), len(runner.Registry.Cases()))
	}
	if err = runner.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// Existing objects in both the same and another tenant must survive every
	// scenario. Their readable fields deliberately overlap the fixture defaults.
	svc := Templates.NewTemplateService(db)
	baselines := []*Template.Template{}
	for _, tenant := range []string{actor.TenantUUID, uuid.NewString()} {
		item, e := svc.Create(Middleware.ContextWithTenantUUID(ctx, tenant), values["name"], values["description"], values["content"])
		if e != nil {
			t.Fatal(e)
		}
		baselines = append(baselines, item)
	}
	ids := []string{}
	for _, definition := range definitions {
		ids = append(ids, definition.UUID)
	}
	reversed := append([]string{}, ids...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	selections := [][]string{ids, reversed}
	for _, id := range ids {
		selections = append(selections, []string{id})
	}
	reports := []Regression.Report{}
	for _, selection := range selections {
		runner, err = Service.NewService(ctx, deps)
		if err != nil {
			t.Fatal(err)
		}
		activeRunner := runner
		t.Cleanup(func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = activeRunner.Close(closeCtx)
		})
		selectedInputs := map[string]any{}
		for _, id := range selection {
			selectedInputs[id] = inputs[id]
		}
		detail, _, e := runner.Create(ctx, actor, Regression.CreateRun{ClientRequestUUID: uuid.NewString(), ProfileUUID: profile, CaseUuids: selection, Inputs: Regression.JSON(selectedInputs)})
		if e != nil {
			t.Fatal(e)
		}
		detail = awaitRegressionRun(t, runner, actor, detail.Run.UUID)
		if err = runner.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if detail.Run.Verdict != "passed" || detail.Run.CleanupState != "succeeded" || len(detail.CaseExecutions) != len(selection) {
			t.Fatalf("%s", Regression.JSON(detail))
		}
		for _, result := range detail.CaseExecutions {
			if result.Verdict != "passed" || result.CleanupState != "succeeded" || len(result.MissingAssertionKeys) != 0 {
				t.Fatalf("%s", Regression.JSON(result))
			}
			for _, fixture := range result.FixtureResources {
				if fixture.CleanupState != "succeeded" || fixture.DisplayName == "" || fixture.RoleKey == "" || fixture.CleanedAt == nil || len(fixture.CleanupEvidence) != 1 {
					t.Fatal(result.CaseUUID, fixture.CleanupState)
				}
				if fixture.CleanupEvidence[0].ResourceUUID != fixture.ResourceUUID || !Regression.Equal(fixture.CleanupEvidence[0].SanitizedData, Regression.JSON(map[string]any{"exists": false})) {
					t.Fatal(fixture.ResourceUUID)
				}
				if fixture.RoleKey == "regression.resourceRoles.copy" {
					ownedSource := false
					for _, source := range result.FixtureResources {
						ownedSource = ownedSource || source.ResourceUUID == fixture.ParentResourceUUID && source.RoleKey == "regression.resourceRoles.source"
					}
					if !ownedSource {
						t.Fatal(fixture.ResourceUUID, fixture.ParentResourceUUID)
					}
				}
			}
			if result.CaseUUID == Scenarios.TemplateInvalidInputUUID {
				for _, step := range result.Steps {
					for _, assertion := range step.Assertions {
						if assertion.Key == "template.rejectedCreate" || assertion.Key == "template.rejectedUpdate" {
							if len(assertion.Evidence) != 1 || assertion.Evidence[0].Boundary != "http" {
								t.Fatal(assertion.Key)
							}
						}
					}
				}
			}
		}
		var remaining []*Template.Template
		if err = db.Find(&remaining).Error; err != nil {
			t.Fatal(err)
		}
		want, actual := []string{}, []string{}
		for _, baseline := range baselines {
			want = append(want, baseline.UUID)
		}
		for _, item := range remaining {
			actual = append(actual, item.UUID)
			if item.Content != values["content"] || item.Name != values["name"] || item.Description != values["description"] {
				t.Fatal(item.UUID)
			}
		}
		sort.Strings(want)
		sort.Strings(actual)
		if !Regression.Equal(want, actual) {
			t.Fatal(want, actual)
		}
		events, e := runner.Store.Events(ctx, actor.TenantUUID, detail.Run.UUID, "")
		if e != nil {
			t.Fatal(e)
		}
		reports = append(reports, Regression.Export(detail, events))
	}
	for _, baseline := range baselines {
		if err = svc.Delete(Middleware.ContextWithTenantUUID(ctx, baseline.TenantUuid), baseline.UUID); err != nil {
			t.Fatal(err)
		}
	}
	var active int64
	if err = db.Model(&Template.Template{}).Count(&active).Error; err != nil || active != 0 {
		t.Fatal(err, active)
	}
	// Fault injection proves that a 400 response alone cannot pass the case:
	// accidental writes are detected, registered and cleaned even on failure.
	t.Run("unexpected_writes_after_http_rejection", func(t *testing.T) {
		faultRunner, e := Service.NewService(ctx, deps)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = faultRunner.Close(ctx) })
		definition, factory, e := faultRunner.Registry.Profile(profile)
		if e != nil {
			t.Fatal(e)
		}
		definition.UUID = uuid.NewString()
		definition.BindingRevision += ".write_fault"
		e = faultRunner.Registry.RegisterProfile(definition, func(ctx context.Context, actor Regression.Actor) (any, []Regression.DependencyEvidence, error) {
			dependency, evidence, err := factory(ctx, actor)
			if err != nil {
				return nil, evidence, err
			}
			business := dependency.(*Scenarios.Dependencies)
			handler := business.TemplateHTTP
			business.TemplateHTTP = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Error(err)
					return
				}
				request.Body.Close()
				request.Body = io.NopCloser(strings.NewReader(string(body)))
				handler.ServeHTTP(w, request)
				if request.Method == http.MethodPost {
					var fields struct{ Name, Description, Content string }
					if err = json.Unmarshal(body, &fields); err != nil {
						t.Error(err)
						return
					}
					if _, err = business.Templates.Create(request.Context(), fields.Name, fields.Description, fields.Content); err != nil {
						t.Error(err)
					}
				}
			})
			return dependency, evidence, nil
		})
		if e != nil {
			t.Fatal(e)
		}
		detail, _, e := faultRunner.Create(ctx, actor, Regression.CreateRun{ClientRequestUUID: uuid.NewString(), ProfileUUID: definition.UUID, CaseUuids: []string{Scenarios.TemplateInvalidInputUUID}, Inputs: Regression.JSON(map[string]any{Scenarios.TemplateInvalidInputUUID: inputs[Scenarios.TemplateInvalidInputUUID]})})
		if e != nil {
			t.Fatal(e)
		}
		detail = awaitRegressionRun(t, faultRunner, actor, detail.Run.UUID)
		if e = faultRunner.Close(ctx); e != nil {
			t.Fatal(e)
		}
		if detail.Run.Verdict != "failed" || detail.Run.CleanupState != "succeeded" || len(detail.CaseExecutions[0].FixtureResources) != 3 {
			t.Fatalf("%s", Regression.JSON(detail))
		}
		failed := []string{}
		for _, step := range detail.CaseExecutions[0].Steps {
			for _, assertion := range step.Assertions {
				if assertion.Status == "failed" {
					failed = append(failed, assertion.Key)
				}
			}
		}
		if !Regression.Equal(failed, []string{"template.noPollution"}) {
			t.Fatal(failed)
		}
		if e = db.Model(&Template.Template{}).Count(&active).Error; e != nil || active != 0 {
			t.Fatal(e, active)
		}
		events, e := faultRunner.Store.Events(ctx, actor.TenantUUID, detail.Run.UUID, "")
		if e != nil {
			t.Fatal(e)
		}
		reports = append(reports, Regression.Export(detail, events))
	})
	if directory := os.Getenv("REGRESSION_EVIDENCE_DIR"); directory != "" {
		if err = os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		evidence := map[string]any{"reports": reports, "case_count": len(definitions), "batch_count": 2, "independent_run_count": 8, "fault_run_count": 1, "active_templates": active, "database": "sqlite", "http_boundary": "in_process_production_routes_trusted_actor"}
		b, err = json.MarshalIndent(evidence, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "template-catalog-evidence.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func awaitRegressionRun(t *testing.T, runner *Regression.Runner, actor Regression.Actor, id string) Regression.RunDetail {
	t.Helper()
	notify, stop := runner.Subscribe(actor.TenantUUID, id)
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for {
		detail, err := runner.Get(ctx, actor, id)
		if err != nil {
			t.Fatal(err)
		}
		if Regression.Terminal(detail.Run.Phase) {
			return detail
		}
		select {
		case <-notify:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}
