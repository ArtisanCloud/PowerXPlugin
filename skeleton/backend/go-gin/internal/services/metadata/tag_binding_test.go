package metadata

import (
	"context"
	"sync"
	"testing"

	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReplaceTagBindingsIsIdempotentAndConcurrent(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:metadata-bindings?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec(`CREATE TABLE metadata_tags (id integer primary key autoincrement, uuid text not null unique, tenant_uuid text not null, namespace text not null, resource_type text not null, code text not null, color text, label_i18n text, description_i18n text, status text not null, usage_count integer not null, created_at datetime, updated_at datetime, deleted_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uk_tags ON metadata_tags(tenant_uuid, namespace, resource_type, code)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE metadata_tag_bindings (id integer primary key autoincrement, uuid text not null unique, tenant_uuid text not null, resource_type text not null, resource_uuid text not null, tag_uuid text not null, created_at datetime, updated_at datetime, deleted_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uk_bindings ON metadata_tag_bindings(tenant_uuid, resource_type, resource_uuid, tag_uuid)`).Error; err != nil {
		t.Fatal(err)
	}
	service := NewService(db)
	ctx := context.Background()
	tag, err := service.CreateTag(ctx, CreateTagInput{TenantUUID: "11111111-1111-4111-8111-111111111111", Namespace: "knowledge", ResourceType: "knowledge.article", Code: "review", LabelI18n: map[string]string{"en": "Review"}})
	if err != nil {
		t.Fatal(err)
	}
	in := ReplaceTagBindingsInput{TenantUUID: "11111111-1111-4111-8111-111111111111", ResourceType: "knowledge.article", ResourceUUID: "22222222-2222-4222-8222-222222222222", TagUUIDs: []string{tag.UUID, tag.UUID}}
	if first, err := service.ReplaceTagBindings(ctx, in); err != nil || len(first) != 1 {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := service.ReplaceTagBindings(ctx, in); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent replace: %v", err)
		}
	}
	bindings, err := service.ListTagBindings(ctx, ListTagBindingsInput{TenantUUID: in.TenantUUID, ResourceType: in.ResourceType, ResourceUUID: in.ResourceUUID})
	if err != nil || len(bindings) != 1 || bindings[0].UUID == "" || bindings[0].TagUUID != tag.UUID {
		t.Fatalf("bindings=%#v err=%v", bindings, err)
	}
	cleared, err := service.ReplaceTagBindings(ctx, ReplaceTagBindingsInput{TenantUUID: in.TenantUUID, ResourceType: in.ResourceType, ResourceUUID: in.ResourceUUID, TagUUIDs: []string{}})
	if err != nil || len(cleared) != 0 {
		t.Fatalf("cleared=%#v err=%v", cleared, err)
	}
}
