package iam

import (
	"time"

	DomainModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/models"
	BaseModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// FederatedExternalIdentity 记录渠道侧外部身份。
type FederatedExternalIdentity struct {
	BaseModels.BaseModel
	Provider       string            `gorm:"size:32;not null;index:idx_fed_ext_provider_user,priority:1" json:"provider"`
	ExternalUserID string            `gorm:"size:128;not null;index:idx_fed_ext_provider_user,priority:2" json:"external_user_id"`
	UnionID        string            `gorm:"size:128;index" json:"union_id"`
	OpenID         string            `gorm:"size:128;index" json:"open_id"`
	Email          string            `gorm:"size:255" json:"email"`
	Phone          string            `gorm:"size:32" json:"phone"`
	TenantScope    string            `gorm:"size:64" json:"tenant_scope"`
	Raw            datatypes.JSONMap `gorm:"type:jsonb" json:"raw"`
}

func (FederatedExternalIdentity) TableName() string {
	return DomainModels.S(DomainModels.TableIAMFederatedExternalIdentities)
}

// FederatedBinding 记录 external identity 与 IAM member 的绑定。
type FederatedBinding struct {
	BaseModels.BaseModel
	Provider       string     `gorm:"size:32;not null;index:idx_fed_bind_provider_user,priority:1" json:"provider"`
	ExternalUserID string     `gorm:"size:128;not null;index:idx_fed_bind_provider_user,priority:2" json:"external_user_id"`
	MemberID       uint64     `gorm:"column:member_id;not null;index" json:"member_id"`
	Status         string     `gorm:"size:32;not null;default:'active';index" json:"status"`
	BoundAt        time.Time  `gorm:"not null;index" json:"bound_at"`
	UnboundAt      *time.Time `gorm:"index" json:"unbound_at,omitempty"`
	Source         string     `gorm:"size:32;not null;default:'jit'" json:"source"`
	MappingVersion string     `gorm:"size:64;not null;default:'v1'" json:"mapping_version"`
}

func (FederatedBinding) TableName() string {
	return DomainModels.S(DomainModels.TableIAMFederatedBindings)
}

// FederatedLoginChallenge 持久化回调 challenge。
type FederatedLoginChallenge struct {
	BaseModels.BaseNoTenantModel
	State      string     `gorm:"size:128;not null;uniqueIndex" json:"state"`
	Nonce      string     `gorm:"size:128;not null" json:"nonce"`
	TenantUUID string     `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	Provider   string     `gorm:"size:32;not null;index" json:"provider"`
	TraceID    string     `gorm:"size:128;index" json:"trace_id"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt *time.Time `gorm:"index" json:"consumed_at,omitempty"`
}

func (FederatedLoginChallenge) TableName() string {
	return DomainModels.S(DomainModels.TableIAMFederatedChallenges)
}

// FederatedRiskEvent 记录风控判定结果。
type FederatedRiskEvent struct {
	BaseModels.BaseNoTenantModel
	TenantUUID   string            `gorm:"type:uuid;not null;index" json:"tenant_uuid"`
	Provider     string            `gorm:"size:32;not null;index" json:"provider"`
	ExternalRef  string            `gorm:"size:160;index" json:"external_ref"`
	RiskType     string            `gorm:"size:64;not null;index" json:"risk_type"`
	Decision     string            `gorm:"size:32;not null;index" json:"decision"`
	ReasonCode   string            `gorm:"size:64;not null;index" json:"reason_code"`
	TraceID      string            `gorm:"size:128;index" json:"trace_id"`
	Evidence     datatypes.JSONMap `gorm:"type:jsonb" json:"evidence"`
	OccurredAt   time.Time         `gorm:"not null;index" json:"occurred_at"`
	BindingID    *uint64           `gorm:"index" json:"binding_id,omitempty"`
	MemberID     *uint64           `gorm:"column:member_id;index" json:"member_id,omitempty"`
	ProviderCode string            `gorm:"size:64" json:"provider_code"`
}

func (FederatedRiskEvent) TableName() string {
	return DomainModels.S(DomainModels.TableIAMFederatedRiskEvents)
}
