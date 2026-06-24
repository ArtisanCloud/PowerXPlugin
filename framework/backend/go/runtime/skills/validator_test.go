package skills

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateManifestRejectsMissingRequiredFields(t *testing.T) {
	m := validManifest()
	m.SkillID = ""
	err := ValidateManifest(m)
	require.Error(t, err)
	require.Contains(t, err.Error(), "skill_id is required")

	m = validManifest()
	m.InputSchema = nil
	err = ValidateManifest(m)
	require.Error(t, err)
	require.Contains(t, err.Error(), "input_schema is required")
}
