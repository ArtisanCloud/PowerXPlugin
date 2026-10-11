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
func (s catalogStub) PresignVariantUpload(context.Context, string, string, powerxmedia.VariantTicketInput) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{URL: s.label}, nil
}
func (s catalogStub) PresignVariantDownload(context.Context, string, string, powerxmedia.VariantTicketInput) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{URL: s.label}, nil
}
func (s catalogStub) CompleteVariantUpload(context.Context, string, string, powerxmedia.CompleteUploadInput) (*powerxmedia.Variant, error) {
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

func TestVariantTransfersUseSelectedAdapter(t *testing.T) {
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		r, err := NewRuntime(mode, catalogStub{label: "local"}, catalogStub{label: "delegated"})
		if err != nil {
			t.Fatal(err)
		}
		s, err := r.Media()
		if err != nil {
			t.Fatal(err)
		}
		u, err := s.PresignVariantUpload(context.Background(), "asset", "variant", powerxmedia.VariantTicketInput{})
		if err != nil || u.URL != string(mode) {
			t.Fatalf("upload=%v err=%v", u, err)
		}
		d, err := s.PresignVariantDownload(context.Background(), "asset", "variant", powerxmedia.VariantTicketInput{})
		if err != nil || d.URL != string(mode) {
			t.Fatalf("download=%v err=%v", d, err)
		}
		v, err := s.CompleteVariantUpload(context.Background(), "asset", "variant", powerxmedia.CompleteUploadInput{})
		if err != nil || v.VariantUUID != string(mode) {
			t.Fatalf("variant=%v err=%v", v, err)
		}
	}
}
