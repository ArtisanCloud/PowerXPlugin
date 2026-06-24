package skills

import "strings"

func ValidateManifest(m PluginSkillManifest) error {
	required := map[string]string{
		"skill_id":    m.SkillID,
		"version":     m.Version,
		"description": m.Description,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return NewError(ErrCodeManifestInvalid, field+" is required")
		}
	}
	if len(m.InputSchema) == 0 {
		return NewError(ErrCodeManifestInvalid, "input_schema is required")
	}
	if strings.TrimSpace(m.Executor.Capability) == "" {
		return NewError(ErrCodeManifestInvalid, "executor.capability is required")
	}
	execType := strings.TrimSpace(m.Executor.Type)
	if execType == "" {
		return NewError(ErrCodeManifestInvalid, "executor.type is required")
	}
	if execType != "capability" {
		return NewError(ErrCodeManifestInvalid, "executor.type must be capability")
	}
	if len(m.Executor.ActionMap) == 0 {
		return NewError(ErrCodeManifestInvalid, "executor.action_map is required")
	}
	return nil
}
