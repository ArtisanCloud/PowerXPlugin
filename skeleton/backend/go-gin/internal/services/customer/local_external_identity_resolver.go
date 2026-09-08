package customer

import (
	"context"
	"errors"
	"strings"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LocalExternalIdentityResolver is the local-mode counterpart of Core's
// external identity resolution capability. Tenant scope comes only from the
// Framework request context; callers cannot supply it in the request DTO.
type LocalExternalIdentityResolver struct {
	db       *gorm.DB
	provider string
}

func NewLocalExternalIdentityResolver(db *gorm.DB, provider string) (*LocalExternalIdentityResolver, error) {
	if db == nil {
		return nil, errors.New("local customer identity resolver requires database")
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, errors.New("local customer identity resolver requires provider namespace")
	}
	return &LocalExternalIdentityResolver{db: db, provider: provider}, nil
}

func (r *LocalExternalIdentityResolver) ResolveExternalIdentity(ctx context.Context, input customerfw.ResolveExternalIdentityRequest) (*customerfw.ExternalIdentityResolution, error) {
	if r == nil || r.db == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer identity resolver unavailable")
	}
	tenantUUID, ok := customerfw.TenantUUIDFromContext(ctx)
	if !ok || tenantUUID == "" {
		return nil, customerfw.NewError(customerfw.CodeCustomerTenantRequired, "customer tenant required")
	}
	subject := strings.TrimSpace(input.ProviderSubject)
	displayName := strings.TrimSpace(input.DisplayName)
	if subject == "" || displayName == "" {
		return nil, customerfw.NewError(customerfw.CodeCustomerIdentitySourceBlocked, "provider_subject and display_name are required")
	}

	var result *customerfw.ExternalIdentityResolution
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var identity customermodel.CustomerAuthIdentity
		err := tx.Where("provider = ? AND provider_subject = ?", r.provider, subject).First(&identity).Error
		switch {
		case err == nil:
			if !isActiveCustomerStatus(identity.Status) {
				return customerfw.NewError(customerfw.CodeCustomerMembershipDisabled, "customer identity is inactive")
			}
			return r.resolveExisting(tx, tenantUUID, identity, &result)
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer identity lookup failed", err)
		}
		return r.createIdentity(tx, tenantUUID, subject, displayName, &result)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LocalExternalIdentityResolver) resolveExisting(tx *gorm.DB, tenantUUID string, identity customermodel.CustomerAuthIdentity, result **customerfw.ExternalIdentityResolution) error {
	var account customermodel.CustomerAccount
	if err := tx.Where("customer_uuid = ?", identity.CustomerUUID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return customerfw.NewError(customerfw.CodeCustomerMembershipRequired, "customer account is missing")
		}
		return customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer account lookup failed", err)
	}
	if !isActiveCustomerStatus(account.Status) {
		return customerfw.NewError(customerfw.CodeCustomerMembershipDisabled, "customer account is inactive")
	}
	membership, err := ensureLocalMembership(tx, tenantUUID, identity.CustomerUUID)
	if err != nil {
		return err
	}
	*result = &customerfw.ExternalIdentityResolution{
		CustomerUUID:   identity.CustomerUUID,
		MembershipUUID: membership.MembershipUUID,
		DisplayName:    strings.TrimSpace(account.DisplayName),
	}
	return nil
}

func (r *LocalExternalIdentityResolver) createIdentity(tx *gorm.DB, tenantUUID, subject, displayName string, result **customerfw.ExternalIdentityResolution) error {
	account := customermodel.CustomerAccount{
		CustomerUUID: uuid.NewString(),
		TenantUuid:   tenantUUID,
		DisplayName:  displayName,
		Status:       customermodel.StatusActive,
		Metadata:     datatypes.JSONMap{},
	}
	if err := tx.Create(&account).Error; err != nil {
		return customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer account create failed", err)
	}
	identity := customermodel.CustomerAuthIdentity{
		CustomerUUID:    account.CustomerUUID,
		Provider:        r.provider,
		ProviderSubject: subject,
		Status:          customermodel.StatusActive,
		Metadata:        datatypes.JSONMap{},
	}
	if err := tx.Create(&identity).Error; err != nil {
		return customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer identity create failed", err)
	}
	membership, err := ensureLocalMembership(tx, tenantUUID, account.CustomerUUID)
	if err != nil {
		return err
	}
	*result = &customerfw.ExternalIdentityResolution{
		CustomerUUID:   account.CustomerUUID,
		MembershipUUID: membership.MembershipUUID,
		DisplayName:    account.DisplayName,
	}
	return nil
}

func ensureLocalMembership(tx *gorm.DB, tenantUUID, customerUUID string) (*customermodel.CustomerTenantMembership, error) {
	var membership customermodel.CustomerTenantMembership
	err := tx.Where("tenant_uuid = ? AND customer_uuid = ?", tenantUUID, customerUUID).First(&membership).Error
	switch {
	case err == nil:
		if !isActiveCustomerStatus(membership.Status) {
			return nil, customerfw.NewError(customerfw.CodeCustomerMembershipDisabled, "customer membership is inactive")
		}
		return &membership, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer membership lookup failed", err)
	}
	membership = customermodel.CustomerTenantMembership{
		MembershipUUID: uuid.NewString(),
		TenantUUID:     tenantUUID,
		CustomerUUID:   customerUUID,
		Status:         customermodel.StatusActive,
		Roles:          datatypes.JSON([]byte("[]")),
		Scopes:         datatypes.JSON([]byte("[]")),
		Source:         string(customerfw.CustomerAuthSourceLocalDev),
		Metadata:       datatypes.JSONMap{},
	}
	if err := tx.Create(&membership).Error; err != nil {
		return nil, customerfw.WrapError(customerfw.CodeCustomerDelegateUnavailable, "local customer membership create failed", err)
	}
	return &membership, nil
}

func isActiveCustomerStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), customermodel.StatusActive)
}
