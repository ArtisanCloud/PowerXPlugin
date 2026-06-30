package skills

type JSONSchema map[string]any

type PluginSkillManifest struct {
	SkillID            string                        `json:"skill_id" yaml:"skill_id"`
	Provider           string                        `json:"provider,omitempty" yaml:"provider,omitempty"`
	Version            string                        `json:"version" yaml:"version"`
	Title              string                        `json:"title,omitempty" yaml:"title,omitempty"`
	Description        string                        `json:"description" yaml:"description"`
	IntentExamples     []string                      `json:"intent_examples,omitempty" yaml:"intent_examples,omitempty"`
	ResponseGuidance   map[string][]string           `json:"response_guidance,omitempty" yaml:"response_guidance,omitempty"`
	ActionRequiredArgs map[string][]string           `json:"action_required_args,omitempty" yaml:"action_required_args,omitempty"`
	ActionOptionalArgs map[string][]string           `json:"action_optional_args,omitempty" yaml:"action_optional_args,omitempty"`
	SlotMapping        map[string]SlotSpec           `json:"slot_mapping,omitempty" yaml:"slot_mapping,omitempty"`
	PendingTaskPolicy  *PendingTaskPolicy            `json:"pending_task_policy,omitempty" yaml:"pending_task_policy,omitempty"`
	StateContract      map[string]any                `json:"state_contract,omitempty" yaml:"state_contract,omitempty"`
	ResultPresentation map[string]ResultPresentation `json:"result_presentation,omitempty" yaml:"result_presentation,omitempty"`
	InputSchema        JSONSchema                    `json:"input_schema" yaml:"input_schema"`
	OutputSchema       JSONSchema                    `json:"output_schema,omitempty" yaml:"output_schema,omitempty"`
	PromptRefs         []string                      `json:"prompt_refs,omitempty" yaml:"prompt_refs,omitempty"`
	Executor           PluginSkillExecutor           `json:"executor" yaml:"executor"`
	Visibility         string                        `json:"visibility,omitempty" yaml:"visibility,omitempty"`
	Status             string                        `json:"status,omitempty" yaml:"status,omitempty"`
}

type SlotSpec struct {
	Labels []string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type PendingTaskPolicy struct {
	Enabled              bool `json:"enabled" yaml:"enabled"`
	MergeWindowMessages  int  `json:"merge_window_messages,omitempty" yaml:"merge_window_messages,omitempty"`
	MergeWindowSeconds   int  `json:"merge_window_seconds,omitempty" yaml:"merge_window_seconds,omitempty"`
	ConfirmBeforeExecute bool `json:"confirm_before_execute,omitempty" yaml:"confirm_before_execute,omitempty"`
}

type ResultPresentation struct {
	Title         string   `json:"title,omitempty" yaml:"title,omitempty"`
	PrimaryLink   string   `json:"primary_link,omitempty" yaml:"primary_link,omitempty"`
	VisibleFields []string `json:"visible_fields,omitempty" yaml:"visible_fields,omitempty"`
}

type PluginSkillExecutor struct {
	Type              string            `json:"type" yaml:"type"`
	Method            string            `json:"method" yaml:"method"`
	Path              string            `json:"path" yaml:"path"`
	Capability        string            `json:"capability" yaml:"capability"`
	PrepareCapability string            `json:"prepare_capability,omitempty" yaml:"prepare_capability,omitempty"`
	ActionMap         map[string]string `json:"action_map,omitempty" yaml:"action_map,omitempty"`
	TimeoutMS         int               `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
	AsyncSupported    bool              `json:"async_supported,omitempty" yaml:"async_supported,omitempty"`
	RiskLevel         string            `json:"risk_level,omitempty" yaml:"risk_level,omitempty"`
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
