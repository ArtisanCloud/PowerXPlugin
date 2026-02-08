package wsbus

import "strings"

func NormalizeAndValidateTopic(topic string) (string, PublishResult) {
	clean := strings.TrimSpace(topic)
	if clean == "" {
		return "", FailureResult(ErrorCodeTopicRequired, "topic is required")
	}
	return NormalizeTopic(clean), SuccessResult()
}

func ValidatePayload(payload any) PublishResult {
	if payload == nil {
		return FailureResult(ErrorCodePayloadRequired, "payload is required")
	}
	return SuccessResult()
}

func ExpandTopicsForRegister(topics []string) ([]string, PublishResult) {
	if len(topics) == 0 {
		return nil, FailureResult(ErrorCodeTopicRequired, "topics is required")
	}
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range topics {
		clean := strings.TrimSpace(raw)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; !ok {
			seen[clean] = struct{}{}
			out = append(out, clean)
		}
		if mapped := NormalizeTopic(clean); mapped != "" && mapped != clean {
			if _, ok := seen[mapped]; !ok {
				seen[mapped] = struct{}{}
				out = append(out, mapped)
			}
		}
	}
	if len(out) == 0 {
		return nil, FailureResult(ErrorCodeTopicRequired, "topics is required")
	}
	return out, SuccessResult()
}
