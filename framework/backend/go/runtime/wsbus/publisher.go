package wsbus

import "context"

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
