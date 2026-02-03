package wsbus

import "strings"

func NormalizeAndValidateTopic(topic string) (string, PublishResult) {
	clean := strings.TrimSpace(topic)
	if clean == "" {
		return "", FailureResult(ErrorCodeTopicRequired, "topic is required")
	}
	if !IsTopicAllowed(clean) {
		return "", FailureResult(ErrorCodeTopicNotAllowed, "topic is not allowed")
	}
	return NormalizeTopic(clean), SuccessResult()
}

func ValidatePayload(payload any) PublishResult {
	if payload == nil {
		return FailureResult(ErrorCodePayloadRequired, "payload is required")
	}
	return SuccessResult()
}
