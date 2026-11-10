package eventfabric

import "context"

func validTopicKeys(keys []string) bool {
	if len(keys) == 0 || len(keys) > 1000 {
		return false
	}
	seen := map[string]bool{}
	for _, key := range keys {
		if !valid(key, 255) || seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

// ResolveTopics uses the selected service credential. Core authenticates its
// tenant, owner, live capability grant and per-topic ACL on every request.
func (c *Client) ResolveTopics(ctx context.Context, in ResolveTopicsRequest) (*ResolvedTopics, error) {
	keys := append([]string(nil), in.TopicKeys...)
	if !identity(in.TenantUUID, in.PluginID) || !validTopicKeys(keys) ||
		(in.CapabilityID != PublishCapability && in.CapabilityID != ConsumeCapability) ||
		(in.CapabilityID == ConsumeCapability && in.SubscriberID != in.PluginID) ||
		(in.CapabilityID == PublishCapability && in.SubscriberID != "") {
		return nil, failure(400, "EVENT_FABRIC_INVALID_ARGUMENT")
	}
	body := map[string]any{"contract_version": Contract, "operation": "resolve_topics", "tenant_uuid": in.TenantUUID, "plugin_id": in.PluginID, "topic_keys": keys}
	if in.CapabilityID == ConsumeCapability {
		body["subscriber_id"] = in.SubscriberID
	}
	var out ResolvedTopics
	if err := c.invoke(ctx, in.CapabilityID, body, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != Contract || out.TenantUUID != in.TenantUUID || out.PluginID != in.PluginID || len(out.TopicBindings) != len(keys) {
		return nil, failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
	}
	physical := map[string]bool{}
	for _, key := range keys {
		full, ok := out.TopicBindings[key]
		if !ok || !physicalTopic(in.TenantUUID, full) || len(full) <= len(in.TenantUUID)+1 || physical[full] {
			return nil, failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
		}
		physical[full] = true
	}
	return &out, nil
}
