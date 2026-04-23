package federated

import (
	"context"
	"fmt"
	"strings"

	providerDingTalk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/dingtalk"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/integration"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const IntegrationTypeIAMFederatedDingTalk = "iam_federated_dingtalk"

type DingTalkConfig struct {
	TenantUUID   string `json:"tenant_uuid"`
	CorpID       string `json:"corp_id"`
	AppKey       string `json:"app_key"`
	AppSecret    string `json:"app_secret"`
	CallbackHost string `json:"callback_host"`
	Status       string `json:"status"`
	SecretID     string `json:"secret_id"`
	RotationDays int    `json:"rotation_days"`
	HttpDebug    bool   `json:"http_debug"`
}

type DingTalkConfigService struct {
	repo *repo.SecretRepository
}

func NewDingTalkConfigService(db *gorm.DB) *DingTalkConfigService {
	if db == nil {
		return &DingTalkConfigService{}
	}
	return &DingTalkConfigService{repo: repo.NewSecretRepository(db)}
}

func (s *DingTalkConfigService) ResolveProviderConfig(ctx context.Context, tenantUUID string) (providerDingTalk.Config, error) {
	cfg, err := s.GetByTenant(ctx, tenantUUID)
	if err != nil {
		return providerDingTalk.Config{}, err
	}
	return providerDingTalk.Config{
		AppKey:      cfg.AppKey,
		AppSecret:   cfg.AppSecret,
		CorpID:      cfg.CorpID,
		CallbackURL: cfg.CallbackHost,
	}, nil
}

func (s *DingTalkConfigService) GetByTenant(ctx context.Context, tenantUUID string) (DingTalkConfig, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return DingTalkConfig{}, fmt.Errorf("tenant_uuid is required")
	}
	if s.repo == nil {
		return DingTalkConfig{}, fmt.Errorf("dingtalk config repository not initialized")
	}
	secret, err := s.repo.GetByIntegrationType(ctx, tenantUUID, IntegrationTypeIAMFederatedDingTalk)
	if err != nil {
		return DingTalkConfig{}, err
	}
	if secret == nil {
		return DingTalkConfig{}, fmt.Errorf("dingtalk config not found")
	}
	status := strings.ToUpper(strings.TrimSpace(secret.Status))
	if status == "REVOKED" {
		return DingTalkConfig{}, fmt.Errorf("dingtalk config is revoked")
	}
	cfg := DingTalkConfig{
		TenantUUID:   tenantUUID,
		SecretID:     secret.ID,
		Status:       status,
		RotationDays: secret.RotationInterval,
		CorpID:       asString(secret.Metadata["corp_id"]),
		AppKey:       firstNonEmpty(asString(secret.Metadata["app_key"]), asString(secret.Metadata["client_id"])),
		AppSecret:    firstNonEmpty(asString(secret.Metadata["app_secret"]), asString(secret.Metadata["client_secret"])),
		CallbackHost: firstNonEmpty(asString(secret.Metadata["callback_host"]), asString(secret.Metadata["callback_url"])),
		HttpDebug:    asBoolDefault(secret.Metadata["http_debug"], false),
	}
	if cfg.CorpID == "" || cfg.AppKey == "" || cfg.AppSecret == "" {
		return DingTalkConfig{}, fmt.Errorf("dingtalk config metadata incomplete")
	}
	return cfg, nil
}

func (s *DingTalkConfigService) UpsertTenantConfig(ctx context.Context, cfg DingTalkConfig) (DingTalkConfig, error) {
	cfg.TenantUUID = strings.TrimSpace(cfg.TenantUUID)
	cfg.CorpID = strings.TrimSpace(cfg.CorpID)
	cfg.AppKey = strings.TrimSpace(cfg.AppKey)
	cfg.AppSecret = strings.TrimSpace(cfg.AppSecret)
	cfg.CallbackHost = strings.TrimSpace(cfg.CallbackHost)
	if cfg.TenantUUID == "" || cfg.CorpID == "" || cfg.AppKey == "" || cfg.AppSecret == "" {
		return DingTalkConfig{}, fmt.Errorf("tenant/corp/app_key/app_secret is required")
	}
	if cfg.RotationDays <= 0 {
		cfg.RotationDays = 30
	}
	if s.repo == nil {
		return DingTalkConfig{}, fmt.Errorf("dingtalk config repository not initialized")
	}
	existing, err := s.repo.GetByIntegrationType(ctx, cfg.TenantUUID, IntegrationTypeIAMFederatedDingTalk)
	if err != nil {
		return DingTalkConfig{}, err
	}
	status := normalizeStatus(cfg.Status)
	meta := datatypes.JSONMap{
		"corp_id":       cfg.CorpID,
		"app_key":       cfg.AppKey,
		"app_secret":    cfg.AppSecret,
		"callback_host": cfg.CallbackHost,
		"http_debug":    cfg.HttpDebug,
	}
	if existing == nil {
		created, createErr := s.repo.Create(ctx, &model.SecretCredential{
			ID:               uuid.NewString(),
			TenantUuid:       cfg.TenantUUID,
			IntegrationType:  IntegrationTypeIAMFederatedDingTalk,
			RotationInterval: cfg.RotationDays,
			Status:           status,
			Metadata:         meta,
		})
		if createErr != nil {
			return DingTalkConfig{}, createErr
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
		return DingTalkConfig{}, err
	}
	cfg.SecretID = existing.ID
	cfg.Status = existing.Status
	return cfg, nil
}
