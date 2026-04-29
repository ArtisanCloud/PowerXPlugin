package integration

import (
	"context"

	domain "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

type noopInvoker struct {
	logger *pxlog.Entry
}

// NewNoopInvoker 返回默认的宿主调用占位实现。
func NewNoopInvoker(logger *pxlog.Entry) HostInvoker {
	return &noopInvoker{logger: logger}
}

func (n *noopInvoker) Invoke(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if n.logger != nil && envelope != nil {
		pxlog.DebugCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":      "integration",
			"biz_scene":   "noop_invoker",
			"biz_domain":  "integration",
			"component":   "integration.noop_invoker",
			"tenant_uuid": envelope.TenantUuid,
			"tool_scope":  envelope.ToolScope,
		}), "noop host invoker executed")
	}
	return &HostInvocationResult{
		Status: "accepted",
	}, nil
}
