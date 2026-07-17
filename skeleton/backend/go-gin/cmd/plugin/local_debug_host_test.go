package main

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestResolveLocalHTTPPort(t *testing.T) {
	tests := []struct {
		name     string
		bindAddr string
		want     int
	}{
		{name: "colon port", bindAddr: ":8078", want: 8078},
		{name: "localhost port", bindAddr: "127.0.0.1:8079", want: 8079},
		{name: "any host port", bindAddr: "0.0.0.0:8080", want: 8080},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveLocalHTTPPort(&config.Config{Server: &config.ServerConfig{BindAddr: tt.bindAddr}})
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestResolveLocalHTTPPortRejectsDynamicPort(t *testing.T) {
	_, err := resolveLocalHTTPPort(&config.Config{Server: &config.ServerConfig{BindAddr: ":0"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive http port")
}

func TestRegisterLocalDebugHostAllowsLocalProxyMode(t *testing.T) {
	t.Setenv("POWERX_PLUGIN_REGISTRATION_MODE", "local")
	t.Setenv("POWERX_PROXY", "1")

	err := registerLocalDebugHostIfNeeded(context.Background(), &config.Config{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "gateway client is required for local plugin registration")
}
