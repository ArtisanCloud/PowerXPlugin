package federated

import (
	"context"
	"errors"
	"strings"
	"time"

	iamrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"gorm.io/gorm"
)

var (
	ErrTenantUUIDRequired   = errors.New("tenant_uuid required")
	ErrProviderRequired     = errors.New("provider required")
	ErrExternalUserRequired = errors.New("external_user_id required")
	ErrMemberInvalid        = errors.New("member invalid for tenant")
)

// BindingService 管理 external identity 与 member 的绑定关系。
type BindingService struct {
	repo       *iamrepo.FederatedBindingRepository
	db         *gorm.DB
	sessionSvc *SessionService
}

type BindInput struct {
	TenantUUID     string
	Provider       string
	ExternalUserID string
	MemberID       uint64
	Source         string
}

func NewBindingService(repo *iamrepo.FederatedBindingRepository, db *gorm.DB, sessionSvc *SessionService) *BindingService {
	return &BindingService{repo: repo, db: db, sessionSvc: sessionSvc}
}

func (s *BindingService) List(ctx context.Context, tenantUUID, provider string) ([]iammodel.FederatedBinding, error) {
	if strings.TrimSpace(tenantUUID) == "" {
		return nil, ErrTenantUUIDRequired
	}
	return s.repo.List(ctx, tenantUUID, provider)
}

func (s *BindingService) Bind(ctx context.Context, in BindInput) (*iammodel.FederatedBinding, error) {
	tenantUUID := strings.TrimSpace(in.TenantUUID)
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	externalUserID := strings.TrimSpace(in.ExternalUserID)
	if tenantUUID == "" {
		return nil, ErrTenantUUIDRequired
	}
	if provider == "" {
		return nil, ErrProviderRequired
	}
	if externalUserID == "" {
		return nil, ErrExternalUserRequired
	}
	if !s.memberInTenant(ctx, in.MemberID, tenantUUID) {
		return nil, ErrMemberInvalid
	}

	existing, err := s.repo.GetActiveByExternal(ctx, tenantUUID, provider, externalUserID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if existing != nil {
		existing.MemberID = in.MemberID
		existing.Source = normalizeSource(in.Source)
		existing.BoundAt = now
		existing.Status = "active"
		existing.UnboundAt = nil
		return s.repo.Save(ctx, existing)
	}

	binding := &iammodel.FederatedBinding{
		BaseModel:      BaseModels(tenantUUID),
		Provider:       provider,
		ExternalUserID: externalUserID,
		MemberID:       in.MemberID,
		Status:         "active",
		BoundAt:        now,
		Source:         normalizeSource(in.Source),
		MappingVersion: "v1",
	}
	return s.repo.Save(ctx, binding)
}

func (s *BindingService) Unbind(ctx context.Context, tenantUUID, provider, externalUserID string) (*iammodel.FederatedBinding, error) {
	if strings.TrimSpace(tenantUUID) == "" {
		return nil, ErrTenantUUIDRequired
	}
	if strings.TrimSpace(provider) == "" {
		return nil, ErrProviderRequired
	}
	if strings.TrimSpace(externalUserID) == "" {
		return nil, ErrExternalUserRequired
	}
	binding, err := s.repo.Unbind(ctx, tenantUUID, provider, externalUserID)
	if err != nil || binding == nil {
		return binding, err
	}
	if s.sessionSvc != nil {
		s.sessionSvc.InvalidateMember(binding.MemberID)
	}
	return binding, nil
}

func (s *BindingService) memberInTenant(ctx context.Context, memberID uint64, tenantUUID string) bool {
	if s == nil || s.db == nil || memberID == 0 {
		return false
	}
	var count int64
	_ = s.db.WithContext(ctx).Model(&iammodel.Member{}).Where("id = ? AND tenant_uuid = ?", memberID, tenantUUID).Count(&count).Error
	return count > 0
}

func normalizeSource(source string) string {
	s := strings.ToLower(strings.TrimSpace(source))
	if s == "" {
		return "admin"
	}
	return s
}

func BaseModels(tenantUUID string) EntityModels.BaseModel {
	return EntityModels.BaseModel{TenantUuid: tenantUUID}
}
