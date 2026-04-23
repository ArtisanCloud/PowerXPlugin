package federated

import (
	"context"
	"strings"
	"testing"

	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDingTalkConfigServiceGetByTenant(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:dingtalk-config-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}
	record := &model.SecretCredential{
		ID:               "secret-a",
		TenantUuid:       "tenant-a",
		IntegrationType:  IntegrationTypeIAMFederatedDingTalk,
		RotationInterval: 30,
		Status:           model.SecretStatusActive,
		Metadata: datatypes.JSONMap{
			"corp_id":    "ding-corp-a",
			"app_key":    "ding-app-key-a",
			"app_secret": "ding-secret-a",
		},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewDingTalkConfigService(db)
	cfg, err := svc.GetByTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("get by tenant: %v", err)
	}
	if cfg.CorpID != "ding-corp-a" || cfg.AppKey != "ding-app-key-a" || cfg.AppSecret != "ding-secret-a" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestDingTalkConfigServiceGetByTenantRejectsRevoked(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:dingtalk-config-revoked?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Create(&model.SecretCredential{
		ID:               "secret-revoked",
		TenantUuid:       "tenant-revoked",
		IntegrationType:  IntegrationTypeIAMFederatedDingTalk,
		RotationInterval: 30,
		Status:           model.SecretStatusRevoked,
		Metadata: datatypes.JSONMap{
			"corp_id":    "ding-corp-r",
			"app_key":    "ding-app-key-r",
			"app_secret": "ding-secret-r",
		},
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewDingTalkConfigService(db)
	_, err = svc.GetByTenant(context.Background(), "tenant-revoked")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "revoked") {
		t.Fatalf("want revoked error, got=%v", err)
	}
}

func TestDingTalkConfigServiceUpsertAndMetadataValidation(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:dingtalk-config-b?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}

	svc := NewDingTalkConfigService(db)
	created, err := svc.UpsertTenantConfig(context.Background(), DingTalkConfig{
		TenantUUID:   "tenant-b",
		Status:       "active",
		RotationDays: 45,
		CorpID:       "ding-corp-b",
		AppKey:       "ding-app-key-b",
		AppSecret:    "ding-secret-b",
		CallbackHost: "https://debug.artisan-cloud.com",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.SecretID == "" {
		t.Fatalf("expected secret id")
	}

	if _, err := svc.GetByTenant(context.Background(), "tenant-b"); err != nil {
		t.Fatalf("expected get by tenant success, err=%v", err)
	}

}

func createIntegrationSecretsTableForConfigTests(db *gorm.DB) error {
	// reduce duplicated DDL across config service tests
	return db.Exec(`CREATE TABLE integration_secrets (
id TEXT PRIMARY KEY,
tenant_uuid TEXT NOT NULL,
integration_type TEXT NOT NULL,
current_secret_ref TEXT,
pending_secret_ref TEXT,
rotation_interval_days INTEGER NOT NULL DEFAULT 30,
last_rotated_at DATETIME,
next_rotation_due_at DATETIME,
status TEXT NOT NULL DEFAULT 'ACTIVE',
audit_log JSON,
	metadata JSON,
	created_at DATETIME,
	updated_at DATETIME
)`).Error
}
