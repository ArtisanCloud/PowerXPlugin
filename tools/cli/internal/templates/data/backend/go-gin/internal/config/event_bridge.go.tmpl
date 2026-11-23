package config

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var ErrEventBridgeRoutingRemoved = errors.New("EVENT_BRIDGE_ROUTING_FIELDS_REMOVED")

// Event routing comes from the resolved provider/proxy matrix, never YAML.
func (c *EventBridgeConfig) UnmarshalYAML(node *yaml.Node) error {
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return err
	}
	for _, name := range []string{"mode", "taskbus_provider", "fallback_to_local"} {
		if _, exists := fields[name]; exists {
			return fmt.Errorf("%w: runtime.event_bridge.%s", ErrEventBridgeRoutingRemoved, name)
		}
	}
	type values EventBridgeConfig
	return node.Decode((*values)(c))
}
