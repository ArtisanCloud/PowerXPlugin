package iamcontext

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
)

// ModeResolutionRecord 记录模式解析输入、结果和审计字段。
type ModeResolutionRecord struct {
	ConfigMode       string                        `json:"config_mode,omitempty"`
	EnvMode          string                        `json:"env_mode,omitempty"`
	EffectiveMode    contracts.IAMMode             `json:"effective_mode"`
	ConflictDetected bool                          `json:"conflict_detected"`
	DecisionReason   string                        `json:"decision_reason"`
	Audit            contracts.ModeResolutionAudit `json:"audit"`
}

func newRecord(configMode, envMode, source, reason, envName string, mode contracts.IAMMode) ModeResolutionRecord {
	return ModeResolutionRecord{
		ConfigMode:     configMode,
		EnvMode:        envMode,
		EffectiveMode:  mode,
		DecisionReason: reason,
		Audit: contracts.ModeResolutionAudit{
			Source:      source,
			ResolvedAt:  time.Now().UTC(),
			Environment: envName,
		},
	}
}
