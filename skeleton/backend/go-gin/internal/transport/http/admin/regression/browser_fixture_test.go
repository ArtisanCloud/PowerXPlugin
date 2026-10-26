package regression

import (
	"context"
	"encoding/json"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Explicitly test-only trusted identity. This is local UI/business evidence, not installed JWT or Core authorization evidence.
func TestRegressionBrowserFixture(t *testing.T) {
	destination := os.Getenv("REGRESSION_BROWSER_FIXTURE")
	if destination == "" {
		t.Skip("REGRESSION_BROWSER_FIXTURE")
	}
	engine, runner, db := fixtureHTTP(t)
	connection, _ := db.DB()
	connection.SetMaxOpenConns(1)
	// Insert the browser-only CORS middleware before the fixture authorization stack.
	wrapped := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,tenant_uuid,X-Tenant-UUID,X-Request-ID,X-Trace-ID,Accept-Language")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/_fixture/reset" && r.Method == "POST" {
			db.Callback().Create().Remove("regression_browser_cancel")
			db.Callback().Update().Remove("regression_browser_fault")
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/_fixture/readback_fault" && r.Method == "POST" {
			input := fixtureInput(t)
			err := db.Callback().Update().After("gorm:update").Register("regression_browser_fault", func(tx *gorm.DB) {
				if tx.Statement.Table == Models.TableTemplate {
					tx.AddError(tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE "+Models.TableTemplate+" SET content = ? WHERE deleted_at IS NULL", input["content"]).Error)
				}
			})
			if err != nil {
				w.WriteHeader(500)
			} else {
				w.WriteHeader(204)
			}
			return
		}
		if r.URL.Path == "/_fixture/cancel_gate" && r.Method == "POST" {
			db.Callback().Update().Remove("regression_browser_fault")
 connection.SetMaxOpenConns(2)
			err := db.Callback().Create().Before("gorm:create").Register("regression_browser_cancel", func(tx *gorm.DB) {
				if tx.Statement.Table == Models.TableTemplate {
					<-tx.Statement.Context.Done()
					tx.AddError(tx.Statement.Context.Err())
				}
			})
			if err != nil {
				w.WriteHeader(500)
			} else {
				w.WriteHeader(204)
			}
			return
		}
		engine.ServeHTTP(w, r)
	}))
	defer wrapped.Close()
	ready, _ := json.Marshal(map[string]any{"api_base": wrapped.URL + "/api/v1", "pid": os.Getpid()})
	if err := os.WriteFile(destination, ready, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(4 * time.Minute)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
wait:
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(destination + ".stop"); err == nil {
				break wait
			}
		case <-deadline.C:
			t.Fatal(Regression.Timeout)
		}
	}
	tenant := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	runs, _, err := runner.Store.List(context.Background(), tenant, 1, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	reports := []Regression.Report{}
	for _, r := range runs {
		d, err := runner.Store.Get(context.Background(), tenant, r.UUID)
		if err != nil {
			t.Fatal(err)
		}
		events, err := runner.Store.Events(context.Background(), tenant, r.UUID, "")
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, Regression.Export(d, events))
	}
	var active int64
	if err = db.Table(Models.S(Models.TableTemplate)).Where("deleted_at IS NULL").Count(&active).Error; err != nil {
		t.Fatal(err)
	}
	evidence, _ := json.MarshalIndent(map[string]any{"reports": reports, "active_templates": active, "authorization_boundary": "injected_test_identity", "database": "sqlite", "api_base": wrapped.URL + "/api/v1"}, "", "  ")
	if err = os.WriteFile(filepath.Join(filepath.Dir(destination), "browser-business-evidence.json"), evidence, 0600); err != nil {
		t.Fatal(err)
	}
}
