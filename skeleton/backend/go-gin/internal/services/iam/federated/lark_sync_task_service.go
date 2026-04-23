package federated

import (
	"context"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/gorm"
)

const topicLarkSyncProgress = "lark.sync.progress"

type LarkSyncTaskService struct {
	base *ChannelSyncTaskService
}

func NewLarkSyncTaskService(repository *repo.ChannelSyncTaskRepository, configSvc *LarkConfigService, publisher fwwsbus.Publisher, db *gorm.DB) *LarkSyncTaskService {
	exec := newSimpleDirectorySyncExecutor(model.ChannelSyncProviderLark, db, func(ctx context.Context, tenantUUID string) (channelSyncConfig, error) {
		cfg, err := configSvc.GetByTenant(ctx, tenantUUID)
		if err != nil {
			return channelSyncConfig{}, err
		}
		return channelSyncConfig{
			TenantScope: strings.TrimSpace(cfg.TenantKey),
			HttpDebug:   cfg.HttpDebug,
			Meta: map[string]any{
				"tenant_key": cfg.TenantKey,
				"app_id":     cfg.AppID,
			},
		}, nil
	})
	return &LarkSyncTaskService{
		base: newChannelSyncTaskService(repository, publisher, model.ChannelSyncProviderLark, topicLarkSyncProgress, exec),
	}
}

func (s *LarkSyncTaskService) Trigger(ctx context.Context, tenantUUID string, action string) (*model.ChannelSyncTask, error) {
	return s.base.Trigger(ctx, tenantUUID, action)
}

func (s *LarkSyncTaskService) List(ctx context.Context, tenantUUID string, limit int) ([]model.ChannelSyncTask, error) {
	return s.base.List(ctx, tenantUUID, limit)
}

func (s *LarkSyncTaskService) Clear(ctx context.Context, tenantUUID string) (int64, error) {
	return s.base.Clear(ctx, tenantUUID)
}
