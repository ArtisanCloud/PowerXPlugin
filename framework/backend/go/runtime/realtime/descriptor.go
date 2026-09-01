package realtime

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Action is the only operation a realtime descriptor can authorize.
type Action string

const (
	ActionPublish   Action = "publish"
	ActionSubscribe Action = "subscribe"
)

// ScopeKind determines the identity fields that must be present for an event.
type ScopeKind string

const (
	ScopeGlobal ScopeKind = "global"
	ScopeTenant ScopeKind = "tenant"
	ScopeMember ScopeKind = "member"
)

// Descriptor is the runtime representation of one events.yaml declaration.
// Keys are declaration templates, for example
// _topic.job.tenant_{{tenant_uuid}}.member_{{member_uuid}}.
type Descriptor struct {
	Key         string     `yaml:"key" json:"key"`
	Protocols   []Protocol `yaml:"protocols" json:"protocols"`
	Actions     []Action   `yaml:"actions" json:"actions"`
	Scope       ScopeKind  `yaml:"scope" json:"scope"`
	EventTypes  []string   `yaml:"event_types" json:"event_types,omitempty"`
	Description string     `yaml:"description" json:"description,omitempty"`
}

type eventsManifest struct {
	Events struct {
		Topics   []Descriptor `yaml:"topics"`
		Channels []Descriptor `yaml:"channels"`
	} `yaml:"events"`
}

// LoadDescriptors parses the realtime descriptor shape from plugin.d/events.yaml.
// Parsing intentionally does not infer protocols, actions, or scope: declarations
// must be complete before they can be used as a runtime authorization allowlist.
func LoadDescriptors(data []byte) ([]Descriptor, error) {
	var manifest eventsManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse realtime events descriptor: %w", err)
	}
	descriptors := append(manifest.Events.Topics, manifest.Events.Channels...)
	if err := ValidateDescriptors(descriptors); err != nil {
		return nil, err
	}
	return descriptors, nil
}

// ValidateDescriptors validates declarations independently from a transport.
func ValidateDescriptors(descriptors []Descriptor) error {
	if len(descriptors) == 0 {
		return errors.New("realtime descriptors are required")
	}
	seen := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		key := strings.TrimSpace(descriptor.Key)
		if !validDescriptorKey(key) {
			return fmt.Errorf("invalid realtime descriptor key %q", descriptor.Key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate realtime descriptor key %q", key)
		}
		seen[key] = struct{}{}
		if err := validateDescriptorValues(descriptor); err != nil {
			return fmt.Errorf("realtime descriptor %q: %w", key, err)
		}
	}
	return nil
}

func validateDescriptorValues(descriptor Descriptor) error {
	protocols := make(map[Protocol]struct{}, len(descriptor.Protocols))
	for _, protocol := range descriptor.Protocols {
		switch protocol {
		case ProtocolSSE, ProtocolWS:
			protocols[protocol] = struct{}{}
		default:
			return fmt.Errorf("unsupported protocol %q", protocol)
		}
	}
	if len(protocols) == 0 {
		return errors.New("protocols are required")
	}
	actions := make(map[Action]struct{}, len(descriptor.Actions))
	for _, action := range descriptor.Actions {
		switch action {
		case ActionPublish, ActionSubscribe:
			actions[action] = struct{}{}
		default:
			return fmt.Errorf("unsupported action %q", action)
		}
	}
	if len(actions) == 0 {
		return errors.New("actions are required")
	}
	switch descriptor.Scope {
	case ScopeGlobal, ScopeTenant, ScopeMember:
	default:
		return fmt.Errorf("unsupported scope %q", descriptor.Scope)
	}
	for _, eventType := range descriptor.EventTypes {
		if strings.TrimSpace(eventType) == "" {
			return errors.New("event_types cannot contain an empty value")
		}
	}
	return nil
}

func validDescriptorKey(key string) bool {
	return strings.HasPrefix(key, "_topic.") ||
		strings.HasPrefix(key, "_channel.") ||
		strings.HasPrefix(key, "powerx.")
}

// PermissionDecision is a deterministic, transport-neutral authorization result.
type PermissionDecision struct {
	Allowed  bool   `json:"allowed"`
	Action   Action `json:"action"`
	Key      string `json:"key"`
	Reason   string `json:"reason"`
	Resource string `json:"resource"`
	TraceID  string `json:"trace_id,omitempty"`
}

// Decide returns deny-by-default authorization for a declared realtime action.
func Decide(descriptors []Descriptor, action Action, key string, protocol Protocol, eventType string, scope Scope) PermissionDecision {
	decision := PermissionDecision{
		Action:   action,
		Key:      strings.TrimSpace(key),
		Resource: "realtime:" + strings.TrimSpace(key),
		TraceID:  strings.TrimSpace(scope.TraceID),
		Reason:   "REALTIME_DESCRIPTOR_NOT_FOUND",
	}
	if action != ActionPublish && action != ActionSubscribe {
		decision.Reason = "REALTIME_ACTION_INVALID"
		return decision
	}
	for _, descriptor := range descriptors {
		if !matchesDescriptorKey(descriptor.Key, decision.Key, scope) {
			continue
		}
		if !hasProtocol(descriptor.Protocols, protocol) {
			decision.Reason = "REALTIME_PROTOCOL_NOT_ALLOWED"
			return decision
		}
		if !hasAction(descriptor.Actions, action) {
			decision.Reason = "REALTIME_ACTION_NOT_ALLOWED"
			return decision
		}
		if !hasEventType(descriptor.EventTypes, eventType) {
			decision.Reason = "REALTIME_EVENT_TYPE_NOT_ALLOWED"
			return decision
		}
		if err := validateScope(descriptor.Scope, scope); err != nil {
			decision.Reason = "REALTIME_SCOPE_INVALID"
			return decision
		}
		decision.Allowed = true
		decision.Reason = "REALTIME_ALLOWED"
		return decision
	}
	return decision
}

// matchesDescriptorKey supports only explicit scope placeholders. It never
// performs wildcard matching, so a descriptor cannot accidentally authorize a
// sibling tenant or member topic.
func matchesDescriptorKey(template, key string, scope Scope) bool {
	template = strings.TrimSpace(template)
	key = strings.TrimSpace(key)
	if template == key {
		return true
	}
	if !strings.Contains(template, "{{") {
		return false
	}
	replacements := map[string]string{
		"{{tenant_uuid}}": strings.TrimSpace(scope.TenantUUID),
		"{{member_uuid}}": strings.TrimSpace(scope.MemberUUID),
	}
	for placeholder, value := range replacements {
		if strings.Contains(template, placeholder) {
			if value == "" {
				return false
			}
			template = strings.ReplaceAll(template, placeholder, value)
		}
	}
	return !strings.Contains(template, "{{") && template == key
}

func hasProtocol(protocols []Protocol, requested Protocol) bool {
	for _, protocol := range protocols {
		if protocol == requested {
			return true
		}
	}
	return false
}

func hasAction(actions []Action, requested Action) bool {
	for _, action := range actions {
		if action == requested {
			return true
		}
	}
	return false
}

func hasEventType(eventTypes []string, requested string) bool {
	if len(eventTypes) == 0 {
		return true
	}
	requested = strings.TrimSpace(requested)
	for _, eventType := range eventTypes {
		if strings.TrimSpace(eventType) == requested {
			return true
		}
	}
	return false
}

func validateScope(kind ScopeKind, scope Scope) error {
	switch kind {
	case ScopeGlobal:
		return nil
	case ScopeTenant:
		if strings.TrimSpace(scope.TenantUUID) == "" {
			return errors.New("tenant_uuid is required")
		}
	case ScopeMember:
		if strings.TrimSpace(scope.TenantUUID) == "" || strings.TrimSpace(scope.MemberUUID) == "" {
			return errors.New("tenant_uuid and member_uuid are required")
		}
	default:
		return fmt.Errorf("unsupported scope %q", kind)
	}
	return nil
}

// DescriptorKeys returns a stable list suitable for diagnostics and tests.
func DescriptorKeys(descriptors []Descriptor) []string {
	keys := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		keys = append(keys, strings.TrimSpace(descriptor.Key))
	}
	sort.Strings(keys)
	return keys
}
