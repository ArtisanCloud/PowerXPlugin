package host_contract

import (
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"reflect"
)

// Status inspects the selected binding without executing a business operation.
// An absent adapter is diagnostic data, never a successful connectivity probe.
func (h *Handler) probeBinding(input map[string]any, result *hostcontract.ProbeResult) error {
	if err := ensureInputKeys(input); err != nil {
		return err
	}
	var err error
	missing := func(v any) error {
		r := reflect.ValueOf(v)
		if !r.IsValid() || (r.Kind() == reflect.Pointer && r.IsNil()) {
			return module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "")
		}
		return nil
	}
	switch result.Module {
	case hostcontract.ModuleIAM:
		err = missing(h.directory)
	case hostcontract.ModuleKnowledge:
		err = missing(h.knowledge)
	case hostcontract.ModuleCache:
		_, err = h.cache.Cache()
	case hostcontract.ModuleTaskCenter:
		_, err = h.tasks.Tasks()
	case hostcontract.ModuleMedia:
		_, err = h.media.Media()
	case hostcontract.ModuleAI:
		_, err = h.ai.Generative()
	case hostcontract.ModuleAgent:
		_, err = h.agent.Lifecycle()
	case hostcontract.ModuleCapabilityRegistry:
		_, err = h.capabilities.Registry()
	case hostcontract.ModuleIntegrationGateway:
		_, err = h.integration.Gateway()
	case hostcontract.ModuleSkills:
		_, err = h.skills.Invoker()
	case hostcontract.ModuleNotifications:
		_, err = h.notifications.Publisher()
	case hostcontract.ModulePluginRuntime:
		_, err = h.pluginRuntime.Service()
	default:
		return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedModule}
	}
	reason := ""
	if err != nil {
		var unavailable *module.Error
		if !errors.As(err, &unavailable) || unavailable.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
			return err
		}
		reason = unavailable.Code
	}
	result.ReasonCode = hostcontract.ReasonCode(reason)
	result.Result = map[string]any{"adapter_available": err == nil, "connectivity_verified": false}
	return nil
}
