package router

import (
	"time"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

// Envelope 定义统一的响应结构。
type Envelope struct {
	Success   bool      `json:"success"`
	Data      any       `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

// APIError 描述错误信息。
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// RespondSuccess 输出成功响应包。
func RespondSuccess(ctx bootstrap.Context, status int, data any, message string) {
	if ctx == nil {
		return
	}
	env := Envelope{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC(),
		RequestID: requestIDFromContext(ctx),
	}
	ctx.JSON(status, env)
}

// RespondError 输出带错误信息的响应包。
func RespondError(ctx bootstrap.Context, status int, code, message string, details any) {
	if ctx == nil {
		return
	}
	env := Envelope{
		Success:   false,
		Error:     &APIError{Code: code, Message: message, Details: details},
		Timestamp: time.Now().UTC(),
		RequestID: requestIDFromContext(ctx),
	}
	ctx.JSON(status, env)
}

func requestIDFromContext(ctx bootstrap.Context) string {
	if ctx == nil {
		return ""
	}
	if id := ctx.Header("X-Request-ID"); id != "" {
		return id
	}
	if id := ctx.Header("Request-ID"); id != "" {
		return id
	}
	return ""
}
