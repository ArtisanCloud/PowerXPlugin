package federated

import (
	"context"
	"fmt"
	"strings"

	providerLark "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/lark"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/integration"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const IntegrationTypeIAMFederatedLark = "iam_federated_lark"

type LarkConfig struct {
	TenantUUID   string `json:"tenant_uuid"`
	TenantKey    string `json:"tenant_key"`
	AppID        string `json:"app_id"`
	AppSecret    string `json:"app_secret"`
	CallbackHost string `json:"callback_host"`
	Status       string `json:"status"`
	SecretID     string `json:"secret_id"`
	RotationDays int    `json:"rotation_days"`
	HttpDebug    bool   `json:"http_debug"`
}

type LarkConfigService struct {
	repo *repo.SecretRepository
}

func NewLarkConfigService(db *gorm.DB) *LarkConfigService {
	if db == nil {
		return &LarkConfigService{}
	}
	return &LarkConfigService{repo: repo.NewSecretRepository(db)}
}

func (s *LarkConfigService) ResolveProviderConfig(ctx context.Context, tenantUUID string) (providerLark.Config, error) {
	cfg, err := s.GetByTenant(ctx, tenantUUID)
	if err != nil {
		return providerLark.Config{}, err
	}
	return providerLark.Config{
		AppID:       cfg.AppID,
		AppSecret:   cfg.AppSecret,
		TenantKey:   cfg.TenantKey,
		CallbackURL: cfg.CallbackHost,
	}, nil
}

func (s *LarkConfigService) GetByTenant(ctx context.Context, tenantUUID string) (LarkConfig, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return LarkConfig{}, fmt.Errorf("tenant_uuid is required")
	}
	if s.repo == nil {
		return LarkConfig{}, fmt.Errorf("lark config repository not initialized")
	}
	secret, err := s.repo.GetByIntegrationType(ctx, tenantUUID, IntegrationTypeIAMFederatedLark)
	if err != nil {
		return LarkConfig{}, err
	}
	if secret == nil {
		return LarkConfig{}, fmt.Errorf("lark config not found")
	}
	status := strings.ToUpper(strings.TrimSpace(secret.Status))
	if status == "REVOKED" {
		return LarkConfig{}, fmt.Errorf("lark config is revoked")
	}
	cfg := LarkConfig{
		TenantUUID:   tenantUUID,
		SecretID:     secret.ID,
		Status:       status,
		RotationDays: secret.RotationInterval,
		TenantKey:    asString(secret.Metadata["tenant_key"]),
		AppID:        firstNonEmpty(asString(secret.Metadata["app_id"]), asString(secret.Metadata["client_id"])),
		AppSecret:    firstNonEmpty(asString(secret.Metadata["app_secret"]), asString(secret.Metadata["client_secret"])),
		CallbackHost: firstNonEmpty(asString(secret.Metadata["callback_host"]), asString(secret.Metadata["callback_url"])),
		HttpDebug:    asBoolDefault(secret.Metadata["http_debug"], false),
	}
	if cfg.TenantKey == "" || cfg.AppID == "" || cfg.AppSecret == "" {
		return LarkConfig{}, fmt.Errorf("lark config metadata incomplete")
	}
	return cfg, nil
}

func (s *LarkConfigService) UpsertTenantConfig(ctx context.Context, cfg LarkConfig) (LarkConfig, error) {
	cfg.TenantUUID = strings.TrimSpace(cfg.TenantUUID)
	cfg.TenantKey = strings.TrimSpace(cfg.TenantKey)
	cfg.AppID = strings.TrimSpace(cfg.AppID)
	cfg.AppSecret = strings.TrimSpace(cfg.AppSecret)
	cfg.CallbackHost = strings.TrimSpace(cfg.CallbackHost)
	if cfg.TenantUUID == "" || cfg.TenantKey == "" || cfg.AppID == "" || cfg.AppSecret == "" {
		return LarkConfig{}, fmt.Errorf("tenant/tenant_key/app_id/app_secret is required")
	}
	if cfg.RotationDays <= 0 {
		cfg.RotationDays = 30
	}
	if s.repo == nil {
		return LarkConfig{}, fmt.Errorf("lark config repository not initialized")
	}
	existing, err := s.repo.GetByIntegrationType(ctx, cfg.TenantUUID, IntegrationTypeIAMFederatedLark)
	if err != nil {
		return LarkConfig{}, err
	}
	status := normalizeStatus(cfg.Status)
	meta := datatypes.JSONMap{
		"tenant_key":    cfg.TenantKey,
		"app_id":        cfg.AppID,
		"app_secret":    cfg.AppSecret,
		"callback_host": cfg.CallbackHost,
		"http_debug":    cfg.HttpDebug,
	}
	if existing == nil {
		created, createErr := s.repo.Create(ctx, &model.SecretCredential{
			ID:               uuid.NewString(),
			TenantUuid:       cfg.TenantUUID,
			IntegrationType:  IntegrationTypeIAMFederatedLark,
			RotationInterval: cfg.RotationDays,
			Status:           status,
			Metadata:         meta,
		})
		if createErr != nil {
			return LarkConfig{}, createErr
		}
		cfg.SecretID = created.ID
		cfg.Status = created.Status
		cfg.RotationDays = created.RotationInterval
		return cfg, nil
	}
	existing.RotationInterval = cfg.RotationDays
	existing.Status = status
	existing.Metadata = meta
	if err := s.repo.Update(ctx, existing); err != nil {
		return LarkConfig{}, err
	}
	cfg.SecretID = existing.ID
	cfg.Status = existing.Status
	return cfg, nil
}
