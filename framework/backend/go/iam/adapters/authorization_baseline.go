package adapters

import (
	"context"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

type validatedBaselines struct {
	pluginID string
	provider contracts.AuthorizationBaselineService
}

// ValidateAuthorizationBaselines wraps the selected provider, including injected
// local stores, without supplying a history implementation or a fallback.
func ValidateAuthorizationBaselines(pluginID string, provider contracts.AuthorizationBaselineService) (contracts.AuthorizationBaselineService, error) {
	if pluginID == "" || strings.TrimSpace(pluginID) != pluginID || nilAdapter(provider) {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, iamerrors.CodeAdapterNotBound)
	}
	return &validatedBaselines{pluginID: pluginID, provider: provider}, nil
}

func (v *validatedBaselines) ReadMemberBaselines(ctx context.Context, query contracts.AuthorizationBaselineQuery) (*contracts.AuthorizationBaselineResult, error) {
	q, err := query.Normalize()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := v.provider.ReadMemberBaselines(ctx, q)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := result.Validate(q, v.pluginID); err != nil {
		return nil, err
	}
	return result, nil
}
