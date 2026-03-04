package wsbus

import "strings"

const (
	TopicTemplateUpdate            = "_topic.template.update"
	TopicAuditTemplateUpdated      = "_topic.audit.template.updated"
	TopicTemplateValidateCompleted = "_topic.template.validate.completed"
	TopicTemplateBatchCloneDone    = "_topic.template.batch_clone.completed"
	TopicTemplateUpdateCompleted   = "_topic.template.update.completed"
)

var allowedTopics = map[string]struct{}{
	TopicTemplateUpdate:            {},
	TopicAuditTemplateUpdated:      {},
	TopicTemplateValidateCompleted: {},
	TopicTemplateBatchCloneDone:    {},
	TopicTemplateUpdateCompleted:   {},
}

var topicAliases = map[string]string{}

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
