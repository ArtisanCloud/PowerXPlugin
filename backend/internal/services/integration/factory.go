package integration

import (
	"time"

	idrepo "github.com/example.com/framework-ci-fixture/backend/internal/entity/repository/integration"
	pxlog "github.com/example.com/framework-ci-fixture/backend/internal/logger"
	"github.com/example.com/framework-ci-fixture/backend/internal/mcp/stream"
	srvtemplates "github.com/example.com/framework-ci-fixture/backend/internal/services/admin/templates"
	"github.com/example.com/framework-ci-fixture/backend/internal/shared/app"
)

// BuildDispatchService 根据应用依赖构造 DispatchService。
func BuildDispatchService(deps *app.Deps, logger *pxlog.Entry) *DispatchService {
	if deps == nil {
		return nil
	}
	if logger == nil {
		logger = pxlog.WithComponent("integration.dispatch_factory")
	}

	loader := NewGrantMatrixLoader(
		deps.DB,
		pxlog.WithComponent("integration.grant_matrix_loader"),
		LoaderOptions{},
	)
	grantService := NewGrantMatrixService(loader, pxlog.WithComponent("integration.grant_matrix_service"))

	var fallbackStore idrepo.IdempotencyProvider
	if deps.DB != nil {
		ttl := 24 * time.Hour
		if deps.Config != nil {
			ttl = deps.Config.IntegrationIdempotencyTTL()
		}
		fallbackStore = idrepo.NewPostgresIdempotencyProvider(deps.DB, ttl)
	}
	repository := idrepo.NewIdempotencyRepository(deps.DB, nil, fallbackStore, pxlog.WithComponent("integration.idempotency_repository"))

	templateService := srvtemplates.NewTemplateService(deps.DB)
	hostLogger := pxlog.WithComponent("integration.host_invoker")
	noopInvoker := NewNoopInvoker(hostLogger)
	invoker := NewCapabilityInvoker(templateService, stream.DefaultBroker(), hostLogger, noopInvoker)

	return NewDispatchService(deps.Config, grantService, repository, invoker, pxlog.WithComponent("integration.dispatch_service"))
}
