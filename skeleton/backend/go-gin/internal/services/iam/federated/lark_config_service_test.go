package federated

import (
	"context"
	"strings"
	"testing"

	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	"gorm.io/datatypes"
)

func TestLarkConfigServiceGetByTenant(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:lark-config-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}
	record := &model.SecretCredential{
		ID:               "secret-a",
		TenantUuid:       "tenant-a",
		IntegrationType:  IntegrationTypeIAMFederatedLark,
		RotationInterval: 30,
		Status:           model.SecretStatusActive,
		Metadata: datatypes.JSONMap{
			"tenant_key": "lark-tenant-a",
			"app_id":     "lark-app-id-a",
			"app_secret": "lark-secret-a",
		},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewLarkConfigService(db)
	cfg, err := svc.GetByTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("get by tenant: %v", err)
	}
	if cfg.TenantKey != "lark-tenant-a" || cfg.AppID != "lark-app-id-a" || cfg.AppSecret != "lark-secret-a" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLarkConfigServiceGetByTenantRejectsRevoked(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:lark-config-revoked?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Create(&model.SecretCredential{
		ID:               "secret-revoked",
		TenantUuid:       "tenant-revoked",
		IntegrationType:  IntegrationTypeIAMFederatedLark,
		RotationInterval: 30,
		Status:           model.SecretStatusRevoked,
		Metadata: datatypes.JSONMap{
			"tenant_key": "lark-tenant-r",
			"app_id":     "lark-app-id-r",
			"app_secret": "lark-secret-r",
		},
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewLarkConfigService(db)
	_, err = svc.GetByTenant(context.Background(), "tenant-revoked")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "revoked") {
		t.Fatalf("want revoked error, got=%v", err)
	}
}

func TestLarkConfigServiceUpsertAndMetadataValidation(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:lark-config-b?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createIntegrationSecretsTableForConfigTests(db); err != nil {
		t.Fatalf("create table: %v", err)
	}

	svc := NewLarkConfigService(db)
	created, err := svc.UpsertTenantConfig(context.Background(), LarkConfig{
		TenantUUID:   "tenant-b",
		Status:       "active",
		RotationDays: 45,
		TenantKey:    "lark-tenant-b",
		AppID:        "lark-app-id-b",
		AppSecret:    "lark-secret-b",
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
