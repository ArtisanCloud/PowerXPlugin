package skills

type JSONSchema map[string]any

type PluginSkillManifest struct {
	SkillID          string              `json:"skill_id"`
	Provider         string              `json:"provider,omitempty"`
	Version          string              `json:"version"`
	Title            string              `json:"title,omitempty"`
	Description      string              `json:"description"`
	IntentExamples   []string            `json:"intent_examples,omitempty"`
	ResponseGuidance map[string][]string `json:"response_guidance,omitempty"`
	InputSchema      JSONSchema          `json:"input_schema"`
	OutputSchema     JSONSchema          `json:"output_schema,omitempty"`
	PromptRefs       []string            `json:"prompt_refs,omitempty"`
	Executor         PluginSkillExecutor `json:"executor"`
	Visibility       string              `json:"visibility,omitempty"`
	Status           string              `json:"status,omitempty"`
}

type PluginSkillExecutor struct {
	Type           string `json:"type"`
	Method         string `json:"method"`
	Path           string `json:"path"`
	Capability     string `json:"capability"`
	ActionMap      map[string]string `json:"action_map,omitempty"`
	TimeoutMS      int    `json:"timeout_ms,omitempty"`
	AsyncSupported bool   `json:"async_supported,omitempty"`
	RiskLevel      string `json:"risk_level,omitempty"`
}

type PluginSkillSchema struct {
	SkillID      string     `json:"skill_id"`
	Version      string     `json:"version"`
	InputSchema  JSONSchema `json:"input_schema"`
	OutputSchema JSONSchema `json:"output_schema,omitempty"`
}

func (m PluginSkillManifest) RegistryKey() string {
	return m.SkillID + "@" + m.Version
}
