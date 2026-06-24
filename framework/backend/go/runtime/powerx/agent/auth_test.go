package agent

import (
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
