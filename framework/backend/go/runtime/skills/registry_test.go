package skills

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func validManifest() PluginSkillManifest {
	return PluginSkillManifest{
		SkillID:     "mediax.video_rebuilder.cn",
		Provider:    "com.powerx.plugins.mediax",
		Version:     "1.0.0",
		Title:       "video rebuild",
		Description: "create video rebuild task",
		InputSchema: JSONSchema{"type": "object"},
		Executor: PluginSkillExecutor{
			Type:       "capability",
			Capability: "creation.video_automation.ingest",
			ActionMap: map[string]string{
				"create": "creation.video_automation.ingest",
			},
		},
	}
}

func TestRegistryRegisterAndList(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterManifest(validManifest()))

	items := reg.List()
	require.Len(t, items, 1)
	require.Equal(t, "mediax.video_rebuilder.cn", items[0].SkillID)
}

func TestRegistryRejectsDuplicateManifest(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterManifest(validManifest()))
	err := reg.RegisterManifest(validManifest())
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrCodeDuplicateManifest)
}

func TestRegistryInvokeDispatchesExecutor(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterManifest(validManifest()))
	require.NoError(t, reg.RegisterExecutor("mediax.video_rebuilder.cn", func(ctx context.Context, inv PluginSkillInvocation) (PluginSkillResult, error) {
		out := SuccessResult(inv, ResultQueued, "queued", nil)
		out.TaskID = "task_001"
		return out, nil
	}))

	result, err := reg.Invoke(context.Background(), validInvocation())
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "task_001", result.TaskID)
	require.Equal(t, "trace_001", result.TraceID)
}
