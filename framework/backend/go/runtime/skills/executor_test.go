package skills

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryInvokeRejectsCapabilityMismatch(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterManifest(validManifest()))
	require.NoError(t, reg.RegisterExecutor("mediax.video_rebuilder.cn", func(ctx context.Context, inv PluginSkillInvocation) (PluginSkillResult, error) {
		return SuccessResult(inv, ResultCompleted, "ok", nil), nil
	}))

	inv := validInvocation()
	inv.Context.Capability = "other.capability"
	_, err := reg.Invoke(context.Background(), inv)
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrCodeCapabilityMismatch)
}
