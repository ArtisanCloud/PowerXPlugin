package provider

import "time"

type ModeResolutionAudit struct {
	Source      string    `json:"source"`
	ResolvedAt  time.Time `json:"resolved_at"`
	Environment string    `json:"environment,omitempty"`
}

type ModeResolutionRecord struct {
	ConfigMode       string              `json:"config_mode,omitempty"`
	EnvMode          string              `json:"env_mode,omitempty"`
	EffectiveMode    Mode                `json:"effective_mode"`
	ConflictDetected bool                `json:"conflict_detected"`
	DecisionReason   string              `json:"decision_reason"`
	Audit            ModeResolutionAudit `json:"audit"`
}

func newRecord(configMode, envMode, source, reason, envName string, mode Mode) ModeResolutionRecord {
	return ModeResolutionRecord{
		ConfigMode:     configMode,
		EnvMode:        envMode,
		EffectiveMode:  mode,
		DecisionReason: reason,
		Audit: ModeResolutionAudit{
			Source:      source,
			ResolvedAt:  time.Now().UTC(),
			Environment: envName,
		},
	}
}
