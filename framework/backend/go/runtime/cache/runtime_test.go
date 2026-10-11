package cache

import (
	"context"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
	"time"
)

type stub struct {
	calls int
	label string
	err   error
}

func (s *stub) Get(context.Context, Scope, GetInput) (*Entry, error) {
	s.calls++
	return &Entry{Found: true, Value: []byte(s.label), ExpiresAt: time.Now().Add(time.Hour)}, s.err
}
func (s *stub) Set(context.Context, Scope, SetInput) error       { s.calls++; return s.err }
func (s *stub) Delete(context.Context, Scope, DeleteInput) error { s.calls++; return s.err }

var scope = Scope{TenantUUID: "00000000-0000-4000-8000-000000000001", Namespace: "license"}

func TestModesAndValidation(t *testing.T) {
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		l, d := &stub{label: "local"}, &stub{label: "delegated"}
		r, err := NewRuntime(mode, l, d)
		if err != nil {
			t.Fatal(err)
		}
		s, err := r.Cache()
		if err != nil {
			t.Fatal(err)
		}
		e, err := s.Get(context.Background(), scope, GetInput{Key: "key"})
		if err != nil || string(e.Value) != string(mode) {
			t.Fatalf("entry=%v err=%v", e, err)
		}
		if err := s.Set(context.Background(), scope, SetInput{Key: "key", TTL: time.Minute}); err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(context.Background(), scope, DeleteInput{Key: "key"}); err != nil {
			t.Fatal(err)
		}
		if mode == provider.ModeLocal && d.calls != 0 || mode == provider.ModeDelegated && l.calls != 0 {
			t.Fatal("mode isolation")
		}
		if err := s.Set(context.Background(), scope, SetInput{Key: "key"}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
		if _, err := s.Get(context.Background(), Scope{}, GetInput{Key: "key"}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
}
func TestUnavailableAndNoFallback(t *testing.T) {
	var missing *stub
	for _, r := range []*Runtime{nil, {}, func() *Runtime { r, _ := NewRuntime(provider.ModeDelegated, &stub{}, missing); return r }()} {
		if _, err := r.Cache(); err == nil {
			t.Fatal("missing adapter accepted")
		}
	}
	l := &stub{}
	failure := errors.New("CACHE_FORBIDDEN")
	r, _ := NewRuntime(provider.ModeDelegated, l, &stub{err: failure})
	s, _ := r.Cache()
	if _, err := s.Get(context.Background(), scope, GetInput{Key: "key"}); !errors.Is(err, failure) || l.calls != 0 {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Delete(ctx, scope, DeleteInput{Key: "key"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
