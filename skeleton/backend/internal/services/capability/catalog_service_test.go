package capability

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	"github.com/stretchr/testify/require"
)

type fakeCapabilitiesManager struct {
	entries []capabilities.CatalogEntry
	err     error
}

func (f *fakeCapabilitiesManager) ListCapabilities(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func (f *fakeCapabilitiesManager) ExportProtocols(ctx context.Context) ([]capabilities.ProtocolAsset, error) {
	return nil, nil
}

func (f *fakeCapabilitiesManager) RegisterWithHost(ctx context.Context, client capabilities.HostSyncClient) error {
	return nil
}

type fakeGatewayClient struct {
	records []gateway.PlatformCapabilityRecord
	err     error
}

func (f *fakeGatewayClient) Enabled() bool { return true }

func (f *fakeGatewayClient) Invoke(ctx context.Context, params gateway.InvokeParams) (*gateway.InvokeResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeGatewayClient) ListPlatformCapabilities(ctx context.Context, opts gateway.ListPlatformCapabilitiesOptions) ([]gateway.PlatformCapabilityRecord, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.records, nil
}

func (f *fakeGatewayClient) Close() error { return nil }

func TestCatalogServiceUsesPlatformCatalogWhenRequested(t *testing.T) {
	manager := &fakeCapabilitiesManager{
		entries: []capabilities.CatalogEntry{
			{ID: "com.demo.local.capability", Version: "0.1.0"},
		},
	}
	gw := &fakeGatewayClient{
		records: []gateway.PlatformCapabilityRecord{
			{
				CapabilityID:  "com.corex.media.assets.read",
				PluginVersion: "1.0.0",
				Protocols: []gateway.PlatformCapabilityProtocol{
					{Channel: "rest", Endpoint: "/api/v1/media/assets", Method: "GET"},
				},
				CapabilitiesHash: "abc",
			},
		},
	}
	svc := &CatalogService{
		manager: manager,
		gateway: gw,
	}
	entries, err := svc.List(context.Background(), ListOptions{Source: "corex"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "com.corex.media.assets.read", entries[0].ID)
	require.Equal(t, "com.corex.media.assets", entries[0].Module)
	require.NotNil(t, entries[0].Protocols["rest"])
}

func TestCatalogServiceFallsBackToLocalWhenPlatformFails(t *testing.T) {
	manager := &fakeCapabilitiesManager{
		entries: []capabilities.CatalogEntry{
			{ID: "com.demo.local.capability", Version: "0.1.0"},
		},
	}
	gw := &fakeGatewayClient{
		err: errors.New("remote unavailable"),
	}
	svc := &CatalogService{
		manager: manager,
		gateway: gw,
	}
	entries, err := svc.List(context.Background(), ListOptions{Source: "corex"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "com.demo.local.capability", entries[0].ID)
}
