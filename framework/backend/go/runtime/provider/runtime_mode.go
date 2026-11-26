package provider

import (
	"errors"
	"strings"
)

var ErrProxyInvalid = errors.New("POWERX_PROXY_INVALID")

// RuntimeMode is the immutable startup decision shared by runtime adapters.
// ProviderMode selects business and event runtimes; ProxyEnabled selects the proxy link.
type RuntimeMode struct {
	ProviderMode Mode
	ProxyEnabled bool
}

func ResolveRuntimeMode(input ResolveInput) (RuntimeMode, ModeResolutionRecord, error) {
	mode, record, err := (ModeResolver{}).Resolve(input)
	if err != nil {
		return RuntimeMode{}, record, err
	}
	proxy := strings.TrimSpace(input.PowerXProxy)
	if proxy != "" && proxy != "0" && proxy != "1" {
		return RuntimeMode{}, record, ErrProxyInvalid
	}
	return RuntimeMode{ProviderMode: mode, ProxyEnabled: proxy == "1"}, record, nil
}

func (m RuntimeMode) Validate() error {
	if m.ProviderMode != ModeLocal && m.ProviderMode != ModeDelegated {
		return ErrModeInvalid
	}
	return nil
}

func (m RuntimeMode) TaskBusProvider() string {
	if m.ProviderMode == ModeDelegated {
		return "host"
	}
	return "redis"
}
