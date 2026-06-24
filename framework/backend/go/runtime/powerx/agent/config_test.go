package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithDefaultsUsesFiveMinuteTimeout(t *testing.T) {
	cfg := PowerXAgentClientConfig{}.WithDefaults()
	require.Equal(t, 5*time.Minute, cfg.Timeout)
}

func TestValidateConfigRequiresBaseURLAndAuth(t *testing.T) {
	err := ValidateConfig(PowerXAgentClientConfig{BearerToken: "token"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url is required")

	err = ValidateConfig(PowerXAgentClientConfig{BaseURL: "https://powerx.example"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bearer token or STS config is required")
}

func TestDelegatedRejectsLegacyTokenEnv(t *testing.T) {
	t.Setenv("PX_TOOL_TOKEN", "legacy")
	err := ValidateConfig(PowerXAgentClientConfig{
		Mode:        ModeDelegated,
		BaseURL:     "https://powerx.example",
		AuthScheme:  AuthBearer,
		BearerToken: "legacy",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PX_TOOL_TOKEN")
}

func TestDelegatedRejectsAnyStaticBearerToken(t *testing.T) {
	err := ValidateConfig(PowerXAgentClientConfig{
		Mode:        ModeDelegated,
		BaseURL:     "https://powerx.example",
		AuthScheme:  AuthBearer,
		BearerToken: "runtime-static-token",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "static bearer token is forbidden")
}
