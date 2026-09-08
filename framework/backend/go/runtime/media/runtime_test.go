package media

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type catalogStub struct{ label string }

func (s catalogStub) ListAssets(context.Context, powerxmedia.ListAssetsInput) (*powerxmedia.ListAssetsOutput, error) {
	if s.label == "error" {
		return nil, errors.New("catalog error")
	}
	return &powerxmedia.ListAssetsOutput{Items: []powerxmedia.Asset{{UUID: s.label}}, Page: 1, PageSize: 1, Total: 1}, nil
}

func (s catalogStub) GetAsset(context.Context, string) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{AssetUUID: s.label}, nil
}
func (s catalogStub) CreateAsset(context.Context, powerxmedia.CreateAssetInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{AssetUUID: s.label}, nil
}
func (s catalogStub) UpdateAsset(context.Context, string, powerxmedia.UpdateAssetInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{AssetUUID: s.label}, nil
}
func (s catalogStub) DeleteAsset(context.Context, string) error { return nil }
func (s catalogStub) PresignUpload(context.Context, string) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (s catalogStub) CompleteUpload(context.Context, string, powerxmedia.CompleteUploadInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{AssetUUID: s.label}, nil
}
func (s catalogStub) PresignDownload(context.Context, string) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (s catalogStub) CreateVariant(context.Context, string, powerxmedia.CreateVariantInput) (*powerxmedia.Variant, error) {
	return &powerxmedia.Variant{VariantUUID: s.label}, nil
}
func (s catalogStub) GetVariant(context.Context, string) (*powerxmedia.Variant, error) {
	return &powerxmedia.Variant{VariantUUID: s.label}, nil
}

func TestRuntimeSelectsOnlyBootstrapMode(t *testing.T) {
	local := catalogStub{label: "local-asset"}
	delegated := catalogStub{label: "delegated-asset"}
	for _, tc := range []struct {
		name string
		mode provider.Mode
		want string
	}{
		{name: "local", mode: provider.ModeLocal, want: "local-asset"},
		{name: "delegated", mode: provider.ModeDelegated, want: "delegated-asset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime, err := NewRuntime(tc.mode, local, delegated)
			if err != nil {
				t.Fatalf("NewRuntime(): %v", err)
			}
			catalog, err := runtime.Assets()
			if err != nil {
				t.Fatalf("Assets(): %v", err)
			}
			result, err := catalog.ListAssets(context.Background(), powerxmedia.ListAssetsInput{})
			if err != nil || len(result.Items) != 1 || result.Items[0].UUID != tc.want {
				t.Fatalf("ListAssets() = %#v, %v", result, err)
			}
		})
	}
}

func TestRuntimeFailsClosedWhenSelectedAdapterIsMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, catalogStub{label: "local"}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	_, err = runtime.Assets()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Assets() error = %v", err)
	}
}
