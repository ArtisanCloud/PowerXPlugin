package iam

import (
	"context"
	"strings"
	"time"

	identitymodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	EntityRepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// FederatedBindingRepository 封装绑定关系的数据访问。
type FederatedBindingRepository struct {
	*EntityRepo.BaseRepository[identitymodel.FederatedBinding]
}

func NewFederatedBindingRepository(db *gorm.DB) *FederatedBindingRepository {
	return &FederatedBindingRepository{BaseRepository: EntityRepo.NewBaseRepository[identitymodel.FederatedBinding](db)}
}

func (r *FederatedBindingRepository) List(ctx context.Context, tenantUUID, provider string) ([]identitymodel.FederatedBinding, error) {
	query := r.DB.WithContext(ctx).Model(&identitymodel.FederatedBinding{}).
		Where("tenant_uuid = ?", strings.TrimSpace(tenantUUID)).
		Order("created_at DESC")
	if strings.TrimSpace(provider) != "" {
		query = query.Where("provider = ?", strings.TrimSpace(provider))
	}
	rows := make([]identitymodel.FederatedBinding, 0)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *FederatedBindingRepository) GetActiveByExternal(ctx context.Context, tenantUUID, provider, externalUserID string) (*identitymodel.FederatedBinding, error) {
	return r.GetActiveByExternalScoped(ctx, tenantUUID, provider, "", externalUserID)
}

func (r *FederatedBindingRepository) GetActiveByExternalScoped(ctx context.Context, tenantUUID, provider, tenantScope, externalUserID string) (*identitymodel.FederatedBinding, error) {
	var row identitymodel.FederatedBinding
	query := r.DB.WithContext(ctx).Model(&identitymodel.FederatedBinding{}).
		Where("tenant_uuid = ? AND provider = ? AND external_user_id = ? AND status = ?", strings.TrimSpace(tenantUUID), strings.TrimSpace(provider), strings.TrimSpace(externalUserID), "active")
	tenantScope = strings.TrimSpace(tenantScope)
	if tenantScope != "" {
		query = query.Where("tenant_scope = ?", tenantScope)
	}
	err := query.First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *FederatedBindingRepository) Save(ctx context.Context, binding *identitymodel.FederatedBinding) (*identitymodel.FederatedBinding, error) {
	if binding == nil {
		return nil, gorm.ErrInvalidData
	}
	if binding.ID == 0 {
		return r.Create(ctx, binding)
	}
	return r.Update(ctx, binding)
}

func (r *FederatedBindingRepository) Unbind(ctx context.Context, tenantUUID, provider, externalUserID string) (*identitymodel.FederatedBinding, error) {
	current, err := r.GetActiveByExternal(ctx, tenantUUID, provider, externalUserID)
	if err != nil || current == nil {
		return current, err
	}
	return r.unbindAndSave(ctx, current)
}

func (r *FederatedBindingRepository) UnbindScoped(ctx context.Context, tenantUUID, provider, tenantScope, externalUserID string) (*identitymodel.FederatedBinding, error) {
	current, err := r.GetActiveByExternalScoped(ctx, tenantUUID, provider, tenantScope, externalUserID)
	if err != nil || current == nil {
		return current, err
	}
	return r.unbindAndSave(ctx, current)
}

func (r *FederatedBindingRepository) unbindAndSave(ctx context.Context, current *identitymodel.FederatedBinding) (*identitymodel.FederatedBinding, error) {
	now := time.Now()
	current.Status = "unbound"
	current.UnboundAt = &now
	return r.Update(ctx, current)
}
