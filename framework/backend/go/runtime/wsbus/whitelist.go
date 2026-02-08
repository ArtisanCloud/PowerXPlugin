package wsbus

import "strings"

const (
	TopicOrgSyncProgress   = "org_sync.progress"
	TopicOrgSyncProgressV1 = "powerx.org_sync.progress.v1"
)

var allowedTopics = map[string]struct{}{
	TopicOrgSyncProgress:   {},
	TopicOrgSyncProgressV1: {},
}

var topicAliases = map[string]string{
	TopicOrgSyncProgress: TopicOrgSyncProgressV1,
}

func IsTopicAllowed(topic string) bool {
	if topic == "" {
		return false
	}
	_, ok := allowedTopics[topic]
	return ok
}

func NormalizeTopic(topic string) string {
	clean := strings.TrimSpace(topic)
	if clean == "" {
		return ""
	}
	if mapped, ok := topicAliases[clean]; ok {
		return mapped
	}
	return clean
}

func TopicAliases() map[string]string {
	out := make(map[string]string, len(topicAliases))
	for k, v := range topicAliases {
		out[k] = v
	}
	return out
}

func AllowedTopics() []string {
	out := make([]string, 0, len(allowedTopics))
	for topic := range allowedTopics {
		out = append(out, topic)
	}
	return out
}
