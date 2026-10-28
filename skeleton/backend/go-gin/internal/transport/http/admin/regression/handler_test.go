package regression

import (
	"context"
	"encoding/json"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	Migrate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/cmd/database/migrate"
	Config "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Template "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	Middleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	Service "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/regression"
	App "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	HTTPMiddleware "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func fixtureHTTP(t *testing.T) (*gin.Engine, *Regression.Runner, *gorm.DB) {
	t.Helper()
	Models.ForceSchemaForTests("")
	t.Cleanup(func() { Models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if err = Migrate.MigrateRegression(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&Template.Template{}); err != nil {
		t.Fatal(err)
	}
	cfg := &Config.Config{DevMode: true, Server: &Config.ServerConfig{Mode: "debug"}, Regression: Config.RegressionConfig{Enabled: true, Environment: "test", Profiles: []Config.RegressionProfileConfig{{UUID: "44444444-4444-4444-8444-444444444444", Key: "controlled", Kind: "controlled", TitleKey: "regression.profiles.controlled", FixtureNamespace: "regression"}}}}
	deps := &App.Deps{DB: db, Config: cfg}
	deps.Regression, err = Service.NewService(context.Background(), deps)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = deps.Regression.Close(ctx)
	})
	engine := gin.New()
	engine.Use(Middleware.RequestID())
	engine.Use(func(c *gin.Context) {
		if c.GetHeader("X-Fixture-Identity") != "missing" {
			tenant := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			if c.GetHeader("X-Fixture-Tenant") != "" {
				tenant = c.GetHeader("X-Fixture-Tenant")
			}
			permissions := []string{"regression.runs:read", "regression.runs:invoke", "regression.runs:manage"}
			if c.GetHeader("X-Fixture-Permissions") == "read" {
				permissions = []string{"regression.runs:read"}
			}
			Middleware.SetTenantContext(c, Middleware.TenantContext{TenantUUID: tenant, UserUUID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", MemberUUID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", Permissions: permissions})
		}
		c.Next()
	})
	engine.Use(HTTPMiddleware.RBAC(&Middleware.RBACConfig{Enabled: true, DefaultDeny: true, RoutePermissions: RBACEntries("/api/v1")}, nil, nil))
	RegisterRoutes(engine.Group("/api/v1/admin"), deps)
	return engine, deps.Regression, db
}
func fixtureInput(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile("../../../../../../../web-admin/nuxt/i18n/locales/en.json")
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	if err = json.Unmarshal(b, &data); err != nil {
		t.Fatal(err)
	}
	var language struct {
		Fixtures struct{ Name, Description, Content, UpdatedContent string } `json:"fixtures"`
	}
	if err = json.Unmarshal(data["regression"], &language); err != nil {
		t.Fatal(err)
	}
	return map[string]any{"name": language.Fixtures.Name, "description": language.Fixtures.Description, "content": language.Fixtures.Content, "updated_content": language.Fixtures.UpdatedContent}
}
func TestRegressionHTTPRealBusinessAndIsolation(t *testing.T) {
	engine, runner, db := fixtureHTTP(t)
	caseUUID := "22222222-2222-4222-8222-222222222222"
	body := map[string]any{"client_request_uuid": uuid.NewString(), "profile_uuid": "44444444-4444-4444-8444-444444444444", "case_uuids": []string{caseUUID}, "inputs": map[string]any{caseUUID: fixtureInput(t)}}
	b, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/regression/runs", strings.NewReader(string(b)))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, request)
	if w.Code != http.StatusAccepted {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var response struct {
		Data Regression.RunDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	a := Regression.Actor{TenantUUID: response.Data.Run.TenantUUID, UUID: response.Data.Run.ActorUUID, RequestID: uuid.NewString(), TraceID: uuid.NewString(), PluginID: App.PluginID}
	notify, stop := runner.Subscribe(a.TenantUUID, response.Data.Run.UUID)
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		d, err := runner.Get(ctx, a, response.Data.Run.UUID)
		if err != nil {
			t.Fatal(err)
		}
		if Regression.Terminal(d.Run.Phase) {
			if d.Run.Verdict != "passed" || d.Run.CleanupState != "succeeded" {
				t.Fatalf("%s %s", d.Run.Verdict, d.CaseExecutions[0].ReasonCode)
			}
			break
		}
		select {
		case <-notify:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	var active int64
	if err := db.Model(&Template.Template{}).Count(&active).Error; err != nil || active != 0 {
		t.Fatal(err, active)
	}
	w = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/regression/runs/"+response.Data.Run.UUID, nil)
	request.Header.Set("X-Fixture-Tenant", uuid.NewString())
	engine.ServeHTTP(w, request)
	if w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/regression/cases", nil)
	request.Header.Set("X-Fixture-Identity", "missing")
	engine.ServeHTTP(w, request)
	if w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
}

func TestRegressionHTTPDedupPermissionsAndEventReplay(t *testing.T) {
	engine, runner, _ := fixtureHTTP(t)
	caseUUID := "22222222-2222-4222-8222-222222222222"
	input := map[string]any{"client_request_uuid": uuid.NewString(), "profile_uuid": "44444444-4444-4444-8444-444444444444", "case_uuids": []string{caseUUID}, "inputs": map[string]any{caseUUID: fixtureInput(t)}}
	send := func(method, path, body, permissions string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if permissions != "" {
			req.Header.Set("X-Fixture-Permissions", permissions)
		}
		engine.ServeHTTP(w, req)
		return w
	}
	b, _ := json.Marshal(input)
	if w := send("POST", "/api/v1/admin/regression/runs", string(b), "read"); w.Code != 403 {
		t.Fatal(w.Code)
	}
	first := send("POST", "/api/v1/admin/regression/runs", string(b), "")
	if first.Code != 202 {
		t.Fatal(first.Body.String())
	}
	var result struct {
		Data Regression.RunDetail `json:"data"`
	}
	_ = json.Unmarshal(first.Body.Bytes(), &result)
	duplicate := send("POST", "/api/v1/admin/regression/runs", string(b), "")
	if duplicate.Code != 202 {
		t.Fatal(duplicate.Body.String())
	}
	var repeated struct {
		Data Regression.RunDetail `json:"data"`
	}
	_ = json.Unmarshal(duplicate.Body.Bytes(), &repeated)
	if repeated.Data.Run.UUID != result.Data.Run.UUID {
		t.Fatal(Regression.RequestConflict)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	a := Regression.Actor{TenantUUID: result.Data.Run.TenantUUID, UUID: result.Data.Run.ActorUUID, PluginID: App.PluginID, RequestID: uuid.NewString(), TraceID: uuid.NewString()}
	notify, stop := runner.Subscribe(a.TenantUUID, result.Data.Run.UUID)
	defer stop()
	for {
		d, err := runner.Get(ctx, a, result.Data.Run.UUID)
		if err != nil {
			t.Fatal(err)
		}
		if Regression.Terminal(d.Run.Phase) {
			break
		}
		select {
		case <-notify:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	input["inputs"] = map[string]any{caseUUID: map[string]any{}}
	b, _ = json.Marshal(input)
	if w := send("POST", "/api/v1/admin/regression/runs", string(b), ""); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := send("POST", "/api/v1/admin/regression/runs", string(b)+"{}", ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
	events, err := runner.Store.Events(ctx, a.TenantUUID, result.Data.Run.UUID, "")
	if err != nil || len(events) < 3 {
		t.Fatal(err)
	}
	cursor := events[0].UUID
	w := send("GET", "/api/v1/admin/regression/runs/"+result.Data.Run.UUID+"/events?after_event_uuid="+cursor, "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "id: "+cursor+"\n") {
		t.Fatal(w.Code)
	}
	for _, e := range events[1:] {
		if !strings.Contains(w.Body.String(), "id: "+e.UUID+"\n") || e.Meta.SourcePlugin != App.PluginID {
			t.Fatal(Regression.EventMetaInvalid)
		}
	}
	if w := send("GET", "/api/v1/admin/regression/runs/"+result.Data.Run.UUID+"/events?after_event_uuid="+uuid.NewString(), "", ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w := send("GET", "/api/v1/admin/regression/runs/"+result.Data.Run.UUID+"/report?format=junit", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "testsuite") {
		t.Fatal(w.Code)
	}
}
