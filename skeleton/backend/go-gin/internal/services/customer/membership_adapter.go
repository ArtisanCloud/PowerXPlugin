package customer

import (
	"context"
	"encoding/json"
	"errors"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	customerrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	"gorm.io/gorm"
)

type FrameworkMembershipResolver struct {
	repo *customerrepo.Repository
}

func NewFrameworkMembershipResolver(db *gorm.DB) *FrameworkMembershipResolver {
	return &FrameworkMembershipResolver{repo: customerrepo.NewRepository(db)}
}

func (r *FrameworkMembershipResolver) Resolve(ctx context.Context, customerUUID string, tenantUUID string) (*customerfw.CustomerMembership, error) {
	if r == nil || r.repo == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	if customerUUID == "" || tenantUUID == "" {
		return nil, customerfw.NewError(customerfw.CodeCustomerMembershipRequired, "customer membership required")
	}
	membership, err := r.repo.ResolveMembership(ctx, customerUUID, tenantUUID)
	if err != nil {
		if errors.Is(err, customerrepo.ErrCustomerNotFound) {
			return nil, customerfw.NewError(customerfw.CodeCustomerMembershipRequired, "customer membership required")
		}
		return nil, err
	}
	return toFrameworkMembership(membership), nil
}

func (r *FrameworkMembershipResolver) List(ctx context.Context, customerUUID string) ([]customerfw.CustomerMembership, error) {
	if r == nil || r.repo == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	if customerUUID == "" {
		return nil, customerfw.NewError(customerfw.CodeCustomerContextMissing, "customer context missing")
	}
	rows, err := r.repo.ListMemberships(ctx, customerUUID)
	if err != nil {
		if errors.Is(err, customerrepo.ErrCustomerNotFound) {
			return nil, customerfw.NewError(customerfw.CodeCustomerMembershipRequired, "customer membership required")
		}
		return nil, err
	}
	out := make([]customerfw.CustomerMembership, 0, len(rows))
	for i := range rows {
		out = append(out, *toFrameworkMembership(&rows[i]))
	}
	return out, nil
}

func toFrameworkMembership(in *customermodel.CustomerTenantMembership) *customerfw.CustomerMembership {
	if in == nil {
		return nil
	}
	return &customerfw.CustomerMembership{
		TenantUUID:     in.TenantUUID,
		CustomerUUID:   in.CustomerUUID,
		MembershipUUID: in.MembershipUUID,
		Status:         customerfw.CustomerMembershipStatus(in.Status),
		Roles:          decodeStringArray(in.Roles),
		Scopes:         decodeStringArray(in.Scopes),
		ExpiresAt:      in.ExpiresAt,
	}
}

func decodeStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
