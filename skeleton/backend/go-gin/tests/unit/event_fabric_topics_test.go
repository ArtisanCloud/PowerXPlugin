package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/security"
)

func TestLoadEventFabricTopics_PreferConfigPath(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "platform_capabilities"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "event_fabric.yaml"), []byte(`
topics:
  - topic: _topic.template.update
  - namespace: _topic.template
    name: batch_clone.completed
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform_capabilities", "event_fabric.yaml"), []byte(`
topics:
  - topic: _topic.should.not.be.used
`), 0o644))

	require.NoError(t, os.Chdir(dir))
	topics, err := security.LoadEventFabricTopics(logrus.NewEntry(logrus.StandardLogger()))
	require.NoError(t, err)
	require.Equal(t, []string{"_topic.template.batch_clone.completed", "_topic.template.update"}, topics)
}
