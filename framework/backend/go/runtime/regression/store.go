package regression

import "context"

type Store interface {
	FindRequest(context.Context, Actor, string) (RunDetail, error)
	Create(context.Context, Actor, RunDetail, RegressionEvent) (RunDetail, bool, error)
	Get(context.Context, string, string) (RunDetail, error)
	List(context.Context, string, uint64, uint64, string) ([]Run, uint64, error)
	Commit(context.Context, string, RunDetail, RegressionEvent) error
	Events(context.Context, string, string, string) ([]RegressionEvent, error)
	Active(context.Context, string) ([]RunDetail, error)
}
