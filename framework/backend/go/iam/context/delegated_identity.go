package iamcontext

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
)

type delegatedIdentityKey struct{}

func WithDelegatedIdentity(ctx context.Context, v *contracts.DelegatedIdentity) context.Context {
	return context.WithValue(ctx, delegatedIdentityKey{}, cloneIdentity(v))
}
func DelegatedIdentityFromContext(ctx context.Context) (*contracts.DelegatedIdentity, bool) {
	if ctx == nil {
		return nil, false
	}
	v, ok := ctx.Value(delegatedIdentityKey{}).(*contracts.DelegatedIdentity)
	return cloneIdentity(v), ok && v != nil
}
func cloneIdentity(v *contracts.DelegatedIdentity) *contracts.DelegatedIdentity {
	if v == nil {
		return nil
	}
	out := *v
	out.Roles = append([]string{}, v.Roles...)
	out.Permissions = append([]string{}, v.Permissions...)
	if v.MemberUUID != nil {
		member := *v.MemberUUID
		out.MemberUUID = &member
	}
	if v.WSRevalidationExpiresAt != nil {
		expiry := *v.WSRevalidationExpiresAt
		out.WSRevalidationExpiresAt = &expiry
	}
	return &out
}
