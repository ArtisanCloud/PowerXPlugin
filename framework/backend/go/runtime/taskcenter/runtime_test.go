package taskcenter

import (
	"context"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

const id = "00000000-0000-4000-8000-000000000001"

type stub struct {
	calls       int
	err         error
	wrongTenant bool
}

func (s *stub) task(scope Scope) (*Task, error) {
	s.calls++
	tenant := scope.TenantUUID
	if s.wrongTenant {
		tenant = "00000000-0000-4000-8000-000000000002"
	}
	return &Task{TaskUUID: id, TenantUUID: tenant, State: Queued, Revision: 1}, s.err
}
func (s *stub) Create(_ context.Context, scope Scope, _ CreateInput) (*Task, error) {
	return s.task(scope)
}
func (s *stub) Get(_ context.Context, scope Scope, _ GetInput) (*Task, error) { return s.task(scope) }
func (s *stub) Update(_ context.Context, scope Scope, in UpdateInput) (*Task, error) {
	out, err := s.task(scope)
	out.State = in.State
	out.Progress = in.Progress
	out.Revision = in.ExpectedRevision + 1
	return out, err
}
func TestModesAndValidation(t *testing.T) {
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		l, d := &stub{}, &stub{}
		r, err := NewRuntime(mode, l, d)
		if err != nil {
			t.Fatal(err)
		}
		s, err := r.Tasks()
		if err != nil {
			t.Fatal(err)
		}
		scope := Scope{TenantUUID: id}
		if _, err := s.Create(context.Background(), scope, CreateInput{Type: "import", IdempotencyKey: "test"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get(context.Background(), scope, GetInput{TaskUUID: id}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Update(context.Background(), scope, UpdateInput{TaskUUID: id, ExpectedRevision: 1, State: Running}); err != nil {
			t.Fatal(err)
		}
		if mode == provider.ModeLocal && d.calls != 0 || mode == provider.ModeDelegated && l.calls != 0 {
			t.Fatal("mode isolation")
		}
		if _, err := s.Get(context.Background(), Scope{}, GetInput{TaskUUID: id}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
		if _, err := s.Update(context.Background(), scope, UpdateInput{TaskUUID: id, State: Succeeded}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
}
func TestUnavailableTenantAndNoFallback(t *testing.T) {
	var missing *stub
	for _, r := range []*Runtime{nil, {}, func() *Runtime { r, _ := NewRuntime(provider.ModeDelegated, &stub{}, missing); return r }()} {
		if _, err := r.Tasks(); err == nil {
			t.Fatal("missing adapter accepted")
		}
	}
	l := &stub{}
	failure := errors.New("TASKCENTER_FORBIDDEN")
	r, _ := NewRuntime(provider.ModeDelegated, l, &stub{err: failure})
	s, _ := r.Tasks()
	if _, err := s.Get(context.Background(), Scope{TenantUUID: id}, GetInput{TaskUUID: id}); !errors.Is(err, failure) || l.calls != 0 {
		t.Fatal(err)
	}
	r, _ = NewRuntime(provider.ModeLocal, &stub{wrongTenant: true}, nil)
	s, _ = r.Tasks()
	if _, err := s.Get(context.Background(), Scope{TenantUUID: id}, GetInput{TaskUUID: id}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Get(ctx, Scope{TenantUUID: id}, GetInput{TaskUUID: id}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
