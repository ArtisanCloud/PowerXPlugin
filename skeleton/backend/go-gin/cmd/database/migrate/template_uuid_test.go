package migrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func templateMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	previous := models.Schema()
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests(previous) })
	db, e := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())}, &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return db
}

func TestTemplateUUIDLegacyMigration(t *testing.T) {
	db := templateMigrationDB(t)
	table := (&model.Template{}).TableName()
	if e := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, tenant_uuid TEXT NOT NULL, name TEXT NOT NULL, deleted_at DATETIME)").Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Exec("INSERT INTO "+table+" (id,tenant_uuid,name,deleted_at) VALUES (1,?,'a',NULL),(2,?,'b','2026-01-01')", uuid.NewString(), uuid.NewString()).Error; e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	if e := ensureTemplateUUIDs(ctx, db); e != nil {
		t.Fatal(e)
	}
	var before []string
	if e := db.Table(table).Order("id").Pluck("uuid", &before).Error; e != nil {
		t.Fatal(e)
	}
	if len(before) != 2 || before[0] == before[1] {
		t.Fatal(before)
	}
	for _, v := range before {
		if _, e := uuid.Parse(v); e != nil {
			t.Fatal(e)
		}
	}
	if e := ensureTemplateUUIDs(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e := db.AutoMigrate(&model.Template{}); e != nil {
		t.Fatal(e)
	}
	var after []string
	if e := db.Table(table).Order("id").Pluck("uuid", &after).Error; e != nil {
		t.Fatal(e)
	}
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("identities changed: %v %v", before, after)
	}
	tpl := model.Template{Name: "new"}
	tpl.TenantUuid = uuid.NewString()
	if e := db.Create(&tpl).Error; e != nil {
		t.Fatal(e)
	}
	if tpl.UUID == "" {
		t.Fatal("missing generated UUID")
	}
	dup := model.Template{UUID: tpl.UUID, Name: "duplicate"}
	dup.TenantUuid = uuid.NewString()
	if e := db.Create(&dup).Error; e == nil {
		t.Fatal("duplicate accepted")
	}
	invalid := model.Template{UUID: "12", Name: "invalid"}
	invalid.TenantUuid = uuid.NewString()
	if e := db.Create(&invalid).Error; e == nil {
		t.Fatal("numeric accepted")
	}
}

func TestTemplateUUIDMigrationRejectsInvalidIdentity(t *testing.T) {
	for _, values := range [][]string{{"", "invalid"}, {uuid.Nil.String(), uuid.NewString()}, {"11111111-1111-4111-8111-111111111111", "11111111-1111-4111-8111-111111111111"}} {
		t.Run(fmt.Sprint(values), func(t *testing.T) {
			db := templateMigrationDB(t)
			table := (&model.Template{}).TableName()
			if e := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, uuid TEXT)").Error; e != nil {
				t.Fatal(e)
			}
			if e := db.Exec("INSERT INTO "+table+" (id,uuid) VALUES (1,?),(2,?)", values[0], values[1]).Error; e != nil {
				t.Fatal(e)
			}
			if e := ensureTemplateUUIDs(context.Background(), db); e == nil {
				t.Fatal("invalid migration accepted")
			}
			var after []string
			db.Table(table).Order("id").Pluck("uuid", &after)
			if fmt.Sprint(after) != fmt.Sprint(values) {
				t.Fatalf("partial rewrite: %v", after)
			}
		})
	}
}
