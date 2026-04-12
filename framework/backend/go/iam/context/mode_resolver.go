package iamcontext

import (
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

const (
	envIAMMode     = "IAM_MODE"
	envPowerXProxy = "POWERX_PROXY"
)

// ModeResolver 基于配置与环境变量解析统一 IAM 模式。
type ModeResolver struct{}

// ResolveInput 描述模式解析输入。
type ResolveInput struct {
	ConfigMode  string
	EnvMode     string
	PowerXProxy string
	Environment string
}

// Resolve 按 “配置优先、冲突 fail-fast” 解析模式。
func (r ModeResolver) Resolve(input ResolveInput) (contracts.IAMMode, ModeResolutionRecord, error) {
	configMode, configSet, err := normalizeMode(input.ConfigMode)
	if err != nil {
		record := newRecord(strings.TrimSpace(input.ConfigMode), strings.TrimSpace(input.EnvMode), "config", "invalid config mode", input.Environment, "")
		return "", record, err
	}

	envRaw := strings.TrimSpace(input.EnvMode)
	if envRaw == "" {
		envRaw = strings.TrimSpace(os.Getenv(envIAMMode))
	}
	powerXProxy := strings.TrimSpace(input.PowerXProxy)
	if powerXProxy == "" {
		powerXProxy = strings.TrimSpace(os.Getenv(envPowerXProxy))
	}

	envMode, envSet, err := normalizeMode(envRaw)
	if err != nil {
		record := newRecord(strings.TrimSpace(input.ConfigMode), envRaw, "env:IAM_MODE", "invalid env mode", input.Environment, "")
		return "", record, err
	}
	if !envSet && powerXProxy == "1" {
		envMode = contracts.IAMModeDelegated
		envSet = true
		envRaw = string(contracts.IAMModeDelegated)
	}

	if configSet && envSet && configMode != envMode {
		record := newRecord(string(configMode), envRaw, "conflict", "config mode conflicts with env mode", input.Environment, "")
		record.ConflictDetected = true
		return "", record, iamerrors.New(iamerrors.CodeModeConflict, "iam mode conflict: config and env differ")
	}

	if configSet {
		record := newRecord(string(configMode), envRaw, "config", "resolved from config.context.iam_mode", input.Environment, configMode)
		return configMode, record, nil
	}
	if envSet {
		source := "env:IAM_MODE"
		if powerXProxy == "1" && strings.TrimSpace(input.EnvMode) == "" && strings.TrimSpace(os.Getenv(envIAMMode)) == "" {
			source = "env:POWERX_PROXY"
		}
		record := newRecord("", envRaw, source, "resolved from environment", input.Environment, envMode)
		return envMode, record, nil
	}

	record := newRecord("", "", "default", "fallback to local mode", input.Environment, contracts.IAMModeLocal)
	return contracts.IAMModeLocal, record, nil
}

func normalizeMode(raw string) (contracts.IAMMode, bool, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "":
		return "", false, nil
	case string(contracts.IAMModeLocal):
		return contracts.IAMModeLocal, true, nil
	case string(contracts.IAMModeDelegated):
		return contracts.IAMModeDelegated, true, nil
	default:
		return "", false, iamerrors.New(iamerrors.CodeModeInvalid, "iam mode must be local or delegated")
	}
}
