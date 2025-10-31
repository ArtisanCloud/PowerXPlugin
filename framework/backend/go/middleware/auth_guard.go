package middleware

import (
	"net/http"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

// AuthGuard 提供默认拒绝行为，提醒业务端显式实现权限校验。
func AuthGuard() bootstrap.Middleware {
	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			ctx.Status(http.StatusNotImplemented)
		}
	}
}
