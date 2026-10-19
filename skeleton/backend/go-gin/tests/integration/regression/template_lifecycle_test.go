package regression

import (
	"context"
	"encoding/json"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Migrate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/cmd/database/migrate"
	Config "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Template "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	Service "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression"
	Scenarios "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression/scenarios"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTemplateRepeatedRunsAndFaultReadback(t *testing.T) {
	Models.ForceSchemaForTests("")
	defer Models.ForceSchemaForTests("public")
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	defer conn.Close()
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
	var locale struct {
		Regression struct {
			Fixtures struct{ Name, Description, Content, UpdatedContent string }
		}
	}
	if err = json.Unmarshal(b, &locale); err != nil {
		t.Fatal(err)
	}
	values := locale.Regression.Fixtures
	profile := uuid.NewString()
	cfg := &Config.Config{DevMode: true, Regression: Config.RegressionConfig{Enabled: true, Environment: "test", Profiles: []Config.RegressionProfileConfig{{UUID: profile, Key: "controlled", Kind: "controlled", TitleKey: "regression.profiles.controlled", FixtureNamespace: "regression"}}}}
	deps := &App.Deps{Config: cfg, DB: db}
	actor := Regression.Actor{TenantUUID: uuid.NewString(), UUID: uuid.NewString(), PluginID: App.PluginID, RequestID: uuid.NewString(), TraceID: uuid.NewString()}
	reports := []Regression.Report{}
	for iteration := 0; iteration < 4; iteration++ {
		if iteration == 3 {
			if err = db.Callback().Update().After("gorm:update").Register("regression_readback_fault", func(tx *gorm.DB) {
				if tx.Statement.Table == Models.TableTemplate {
					if e := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE "+Models.TableTemplate+" SET content = ? WHERE tenant_uuid = ? AND deleted_at IS NULL", values.Content, actor.TenantUUID).Error; e != nil {
						tx.AddError(e)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
		}
		r, err := Service.NewService(ctx, deps)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = r.Close(closeCtx)
		})
		d, _, err := r.Create(ctx, actor, Regression.CreateRun{ClientRequestUUID: uuid.NewString(), ProfileUUID: profile, CaseUuids: []string{Scenarios.TemplateCaseUUID}, Inputs: Regression.JSON(map[string]any{Scenarios.TemplateCaseUUID: map[string]any{"name": values.Name, "description": values.Description, "content": values.Content, "updated_content": values.UpdatedContent}})})
		if err != nil {
			t.Fatal(err)
		}
		notify, stop := r.Subscribe(actor.TenantUUID, d.Run.UUID)
		timeout, cancel := context.WithTimeout(ctx, time.Second*30)
		for {
			d, err = r.Get(timeout, actor, d.Run.UUID)
			if err != nil {
				t.Fatal(err)
			}
			if Regression.Terminal(d.Run.Phase) {
				break
			}
			select {
			case <-notify:
			case <-timeout.Done():
				t.Fatal(timeout.Err())
			}
		}
		stop()
		cancel()
		if err = r.Close(ctx); err != nil {
			t.Fatal(err)
		}
		expected := "passed"
		if iteration == 3 {
			expected = "failed"
		}
		if d.Run.Verdict != expected || d.Run.CleanupState != "succeeded" {
			t.Fatal(d.Run)
		}
		var active int64
		db.Model(&Template.Template{}).Where("tenant_uuid = ?", actor.TenantUUID).Count(&active)
		if active != 0 {
			t.Fatal(active)
		}
		events, err := r.Store.Events(ctx, actor.TenantUUID, d.Run.UUID, "")
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, Regression.Export(d, events))
	}
	if directory := os.Getenv("REGRESSION_EVIDENCE_DIR"); directory != "" {
		if err = os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		b, err = json.MarshalIndent(reports, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "controlled-template-reports.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
