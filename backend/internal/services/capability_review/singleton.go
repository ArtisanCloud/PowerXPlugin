package capability_review

import (
	"sync"

	"github.com/example.com/framework-ci-fixture/backend/internal/shared/app"
)

var (
	sharedOnce sync.Once
	sharedSvc  *WorkflowService
)

// SharedWorkflowService returns a process-wide singleton instance backed by WorkflowService.
func SharedWorkflowService(deps *app.Deps) *WorkflowService {
	sharedOnce.Do(func() {
		sharedSvc = NewWorkflowService(deps)
	})
	return sharedSvc
}
