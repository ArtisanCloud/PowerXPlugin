// Package guideexamples contains executable documentation, not a local store.
package guideexamples

import (
	"context"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	corecap "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	corenotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// BuildPublisher belongs in bootstrap. mode and required come from trusted,
// validated configuration/manifest; tokens comes from the host STS provider.
// local is the plugin's real implementation, not a Framework-created store.
func BuildPublisher(ctx context.Context, mode provider.Mode, local notifications.Publisher,
	baseURL string, tokens func(context.Context) (string, error), required []string,
	httpClient *http.Client,
) (notifications.Publisher, error) {
	var delegated notifications.Publisher
	if mode == provider.ModeDelegated {
		if tokens == nil {
			return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "sts.token_provider")
		}
		checker, err := corecap.NewClientWithTokenProvider(corecap.Config{BaseURL: baseURL}, corecap.TokenProviderFunc(tokens), httpClient)
		if err != nil {
			return nil, err
		}
		if err := capability.RequireGrants(ctx, checker, required); err != nil {
			return nil, err
		}
		delegated, err = corenotifications.NewClientWithTokenProvider(corenotifications.Config{BaseURL: baseURL}, corenotifications.TokenProviderFunc(tokens), httpClient)
		if err != nil {
			return nil, err
		}
	}
	runtime, err := notifications.NewRuntime(mode, local, delegated)
	if err != nil {
		return nil, err
	}
	return runtime.Publisher()
}

// ValidateCustomer runs at a request boundary. rawCustomerJWT is never a tenant
// selector or a service token. Core validates it; this helper does not trust it.
func ValidateCustomer(ctx context.Context, runtime *customerfw.Runtime, rawCustomerJWT string) (*customerfw.CustomerContext, error) {
	auth, err := runtime.Auth()
	if err != nil {
		return nil, err
	}
	ctx = customerfw.WithCustomerCredential(ctx, rawCustomerJWT)
	return auth.Validate(ctx, rawCustomerJWT)
}
