package capability

import (
	"context"
	"reflect"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

type GrantChecker interface {
	GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error)
}

// RequireGrants checks only explicitly required capabilities. Optional modules
// remain callable and return their own authorization errors. This is a startup
// check, not a replacement for Core authorization on every business request.
func RequireGrants(ctx context.Context, checker GrantChecker, required []string) error {
	seen := map[string]bool{}
	for _, id := range required {
		if id == "" || strings.TrimSpace(id) != id || seen[id] {
			return module.NewError("FRAMEWORK_CAPABILITY_REQUIREMENTS_INVALID", id)
		}
		seen[id] = true
	}
	if len(required) == 0 {
		return nil
	}
	if nilGrantChecker(checker) {
		return module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "capability.grant_status")
	}
	for start := 0; start < len(required); start += 100 {
		end := min(start+100, len(required))
		ids := append([]string(nil), required[start:end]...)
		items, err := checker.GrantStatus(ctx, powerxcapability.GrantStatusInput{CapabilityIDs: ids})
		if err != nil {
			return err
		}
		if len(items) != len(ids) {
			return module.NewError("FRAMEWORK_CAPABILITY_GRANT_RESPONSE_INVALID", "capability.grant_status")
		}
		for i, item := range items {
			if item.CapabilityID != ids[i] || item.ReasonCode == "" {
				return module.NewError("FRAMEWORK_CAPABILITY_GRANT_RESPONSE_INVALID", ids[i])
			}
			switch item.Status {
			case "granted":
			case "not_granted", "unknown":
				return module.NewError("FRAMEWORK_REQUIRED_CAPABILITY_UNAVAILABLE", item.CapabilityID+":"+item.ReasonCode)
			default:
				return module.NewError("FRAMEWORK_CAPABILITY_GRANT_RESPONSE_INVALID", ids[i])
			}
		}
	}
	return nil
}

func nilGrantChecker(checker GrantChecker) bool {
	if checker == nil {
		return true
	}
	value := reflect.ValueOf(checker)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return value.IsNil()
	default:
		return false
	}
}
