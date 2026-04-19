package federated

import (
	"context"
	"testing"

	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	"gorm.io/datatypes"
)

func TestWeComConfigServiceGetByTenant(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:wecom-config-a?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE integration_secrets (
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
)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	record := &model.SecretCredential{
		ID:               "secret-a",
		TenantUuid:       "tenant-a",
		IntegrationType:  IntegrationTypeIAMFederatedWeCom,
		RotationInterval: 30,
		Status:           model.SecretStatusActive,
		Metadata: datatypes.JSONMap{
			"corp_id":          "wx-corp-a",
			"agent_id":         100001,
			"app_secret":       "sec-a",
			"token":            "tok-a",
			"encoding_aes_key": "aes-a",
		},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewWeComConfigService(db)
	cfg, err := svc.GetByTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("get by tenant: %v", err)
	}
	if cfg.CorpID != "wx-corp-a" || cfg.AgentID != 100001 || cfg.Secret != "sec-a" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestWeComConfigServiceUpsertAndResolveCallback(t *testing.T) {
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })

	db, err := openTestSQLite("file:wecom-config-b?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE integration_secrets (
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
)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	svc := NewWeComConfigService(db)
	created, err := svc.UpsertTenantConfig(context.Background(), WeComConfig{
		TenantUUID:   "tenant-b",
		Status:       "active",
		RotationDays: 45,
		CallbackHost: "https://debug.artisan-cloud.com",
		CorpID:       "wx-corp-b",
		AgentID:      100009,
		Secret:       "sec-b",
		Token:        "tok-b",
		AESKey:       "aes-b",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.SecretID == "" {
		t.Fatalf("expected secret id")
	}

	cbCfg, err := svc.ResolveCallbackConfig(context.Background(), "tenant-b", "wx-corp-b", "100009")
	if err != nil {
		t.Fatalf("resolve callback: %v", err)
	}
	if cbCfg.Token != "tok-b" || cbCfg.AESKey != "aes-b" {
		t.Fatalf("unexpected callback config: %+v", cbCfg)
	}

	if _, err := svc.ResolveCallbackConfig(context.Background(), "tenant-b", "wx-corp-b", "100010"); err == nil {
		t.Fatalf("expected app_id mismatch error")
	}
}
