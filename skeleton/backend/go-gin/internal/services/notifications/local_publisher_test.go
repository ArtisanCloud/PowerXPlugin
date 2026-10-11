package notifications

import (
	"context"
	"testing"

	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	marketplacemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/marketplace"
	"gorm.io/gorm"
)

func TestLocalPublisherUsesTrustedTenantAndPersistsNotification(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(dbx.SQLiteDialector("file:local_notification_publisher?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE marketplace_notifications (id TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, recipient_type TEXT NOT NULL, recipient_id TEXT NOT NULL, channel TEXT NOT NULL, template_code TEXT NOT NULL, payload TEXT, scheduled_at DATETIME, sent_at DATETIME, status TEXT NOT NULL, created_at DATETIME, updated_at DATETIME);`).Error; err != nil {
		t.Fatalf("create notification table: %v", err)
	}
	publisher, err := NewLocalPublisher(db, func(context.Context) (string, bool) { return "tenant-trusted", true })
	if err != nil {
		t.Fatalf("NewLocalPublisher(): %v", err)
	}
	created, err := publisher.Create(context.Background(), powerxnotifications.CreateInput{Title: "Title", Content: "Content", MemberUUID: "member-1", Type: "notice", Metadata: map[string]any{"source": "test"}})
	if err != nil {
		t.Fatalf("Create(): %v", err)
	}
	if created.UUID == "" || created.MemberUUID != "member-1" {
		t.Fatalf("unexpected response: %#v", created)
	}
	var record marketplacemodel.Notification
	if err := db.Where("id = ?", created.UUID).First(&record).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if record.TenantUuid != "tenant-trusted" || record.RecipientID != "member-1" {
		t.Fatalf("record must use trusted tenant and member target: %#v", record)
	}
}

func TestLocalPublisherRejectsMissingTrustedTenant(t *testing.T) {
	publisher := &LocalPublisher{db: &gorm.DB{}, resolveTenant: func(context.Context) (string, bool) { return "", false }}
	_, err := publisher.Create(context.Background(), powerxnotifications.CreateInput{Title: "Title", Content: "Content"})
	if err == nil {
		t.Fatal("Create() error = nil, want tenant rejection")
	}
}
