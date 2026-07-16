package provider

import (
	"os"
	"strings"
)

const EnvProviderMode = "POWERX_PROVIDER_MODE"

type ModeResolver struct{}

type ResolveInput struct {
	ConfigMode  string
	EnvMode     string
	PowerXProxy string
	Environment string
}

func (r ModeResolver) Resolve(input ResolveInput) (Mode, ModeResolutionRecord, error) {
	configMode, configSet, err := NormalizeMode(input.ConfigMode)
	if err != nil {
		record := newRecord(strings.TrimSpace(input.ConfigMode), strings.TrimSpace(input.EnvMode), "config", "invalid config mode", input.Environment, "")
		return "", record, err
	}

	envRaw := strings.TrimSpace(input.EnvMode)
	if envRaw == "" {
		envRaw = strings.TrimSpace(os.Getenv(EnvProviderMode))
	}
	envMode, envSet, err := NormalizeMode(envRaw)
	if err != nil {
		record := newRecord(strings.TrimSpace(input.ConfigMode), envRaw, "env:POWERX_PROVIDER_MODE", "invalid env provider mode", input.Environment, "")
		return "", record, err
	}

	if configSet && envSet && configMode != envMode {
		record := newRecord(string(configMode), envRaw, "conflict", "config mode conflicts with env mode", input.Environment, "")
		record.ConflictDetected = true
		return "", record, ErrModeConflict
	}

	if configSet {
		record := newRecord(string(configMode), envRaw, "config", "resolved from config.context.provider_mode", input.Environment, configMode)
		return configMode, record, nil
	}
	if envSet {
		record := newRecord("", envRaw, "env:POWERX_PROVIDER_MODE", "resolved from environment", input.Environment, envMode)
		return envMode, record, nil
	}

	record := newRecord("", "", "default", "fallback to local mode", input.Environment, ModeLocal)
	return ModeLocal, record, nil
}
