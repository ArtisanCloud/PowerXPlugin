// Package media defines the Framework-owned, mode-bound media catalog
// boundary. Plugin business code depends on Runtime, never on a local store or
// a PowerX capability client directly.
package media

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// AssetCatalog is retained for read-only consumers.
type AssetCatalog interface {
	ListAssets(context.Context, powerxmedia.ListAssetsInput) (*powerxmedia.ListAssetsOutput, error)
}

// Service is the full currently-published Media Host Contract. Plugins inject
// their own local implementation; delegated mode uses powerxmedia.HostClient.
type Service interface {
	AssetCatalog
	GetAsset(context.Context, string) (*powerxmedia.HostAsset, error)
	CreateAsset(context.Context, powerxmedia.CreateAssetInput) (*powerxmedia.HostAsset, error)
	UpdateAsset(context.Context, string, powerxmedia.UpdateAssetInput) (*powerxmedia.HostAsset, error)
	DeleteAsset(context.Context, string) error
	PresignUpload(context.Context, string) (*powerxmedia.TransferTicket, error)
	CompleteUpload(context.Context, string, powerxmedia.CompleteUploadInput) (*powerxmedia.HostAsset, error)
	PresignDownload(context.Context, string) (*powerxmedia.TransferTicket, error)
	CreateVariant(context.Context, string, powerxmedia.CreateVariantInput) (*powerxmedia.Variant, error)
	GetVariant(context.Context, string) (*powerxmedia.Variant, error)
}

// Runtime selects one startup-supplied adapter from the trusted ProviderMode.
// It does not inspect requests or construct a local/delegated fallback.
type Runtime struct {
	mode    provider.Mode
	service *module.Factory[Service]
}

// NewRuntime creates a fail-closed media runtime. local and delegated are
// plugin/bootstrap supplied implementations of the same contract.
func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	service, err := module.NewFactory("media.service", mode,
		module.Binding[Service]{Value: local, Available: local != nil},
		module.Binding[Service]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, service: service}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

// Assets returns the adapter selected at bootstrap.
func (r *Runtime) Assets() (AssetCatalog, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "media runtime is unavailable")
	}
	service, err := r.service.Resolve()
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (r *Runtime) Media() (Service, error) {
	if r == nil || r.service == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "media runtime is unavailable")
	}
	return r.service.Resolve()
}

var _ Service = (*powerxmedia.HostClient)(nil)
