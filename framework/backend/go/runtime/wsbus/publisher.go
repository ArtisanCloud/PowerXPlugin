package wsbus

import "context"

const (
	ErrorCodeTopicRequired           = "TOPIC_REQUIRED"
	ErrorCodeTopicNotAllowed         = "TOPIC_NOT_ALLOWED"
	ErrorCodePayloadRequired         = "PAYLOAD_REQUIRED"
	ErrorCodeTenantRequired          = "TENANT_REQUIRED"
	ErrorCodePublisherNotConfigured  = "PUBLISHER_NOT_CONFIGURED"
	ErrorCodeHostPublishFailed       = "HOST_PUBLISH_FAILED"
	ErrorCodeLocalPublishFailed      = "LOCAL_PUBLISH_FAILED"
	ErrorCodePublishRequestInvalid   = "PUBLISH_REQUEST_INVALID"
	ErrorCodePublishResponseInvalid  = "PUBLISH_RESPONSE_INVALID"
	ErrorCodePublishUpstreamRejected = "PUBLISH_UPSTREAM_REJECTED"
)

type PublishOptions struct {
	TenantUUID string `json:"tenant_uuid"`
	TraceID    string `json:"trace_id"`
}

type PublishResult struct {
	OK           bool   `json:"ok"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload any, opts PublishOptions) PublishResult
}

func SuccessResult() PublishResult {
	return PublishResult{OK: true}
}

func FailureResult(code, message string) PublishResult {
	return PublishResult{OK: false, ErrorCode: code, ErrorMessage: message}
}
