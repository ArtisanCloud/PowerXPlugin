package integration

import pxlog "github.com/example.com/framework-ci-fixture/backend/internal/logger"

// Dependencies aggregates shared dependencies for integration services.
type Dependencies struct {
	Logger *pxlog.Entry
}

// NewDependencies constructs a new dependency container with sane defaults.
func NewDependencies(logger *pxlog.Entry) *Dependencies {
	return &Dependencies{Logger: logger}
}
