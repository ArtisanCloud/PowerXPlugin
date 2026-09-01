package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelegatedAllowsSTSConfig(t *testing.T) {
	err := ValidateConfig(PowerXAgentClientConfig{
		Mode:            ModeDelegated,
		BaseURL:         "https://powerx.example",
		AuthScheme:      AuthBearer,
		STSClientID:     "client",
		STSClientSecret: "secret",
		STSTokenURL:     "https://powerx.example/sts",
	})
	require.NoError(t, err)
}

func TestDelegatedClientAcceptsOnlyInjectedSTSProvider(t *testing.T) {
	client, err := NewClientWithTokenProvider(PowerXAgentClientConfig{
		Mode:    ModeDelegated,
		BaseURL: "https://core.example",
	}, TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil }))
	if err != nil || client == nil {
		t.Fatalf("delegated injected provider: client=%v err=%v", client, err)
	}
	_, err = NewClientWithTokenProvider(PowerXAgentClientConfig{Mode: ModeDelegated, BaseURL: "https://core.example", BearerToken: "forbidden"}, TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil }))
	if err == nil {
		t.Fatal("delegated static bearer must be rejected")
	}
}
