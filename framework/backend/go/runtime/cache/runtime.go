// Package cache defines tenant-scoped cache access. Plugins own local storage;
// the runtime selects exactly one startup-supplied adapter, never a fallback.
package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/google/uuid"
)

var (
	ErrInvalidArgument = errors.New("CACHE_INVALID_ARGUMENT")
	ErrInvalidResponse = errors.New("CACHE_INVALID_RESPONSE")
)

// Scope must come from trusted startup/request identity, not request JSON.
// Adapters must isolate tenant + namespace + key. A delegated adapter must
// additionally use credential-derived tenant/plugin authority on the host.
type Scope struct {
	TenantUUID string
	Namespace  string
}
type GetInput struct{ Key string }
type SetInput struct {
	Key   string
	Value []byte
	TTL   time.Duration
}
type DeleteInput struct{ Key string }

// Found distinguishes a cache miss from a stored empty value. Dependency
// failures are errors, never misses. A miss must have no value or expiry.
type Entry struct {
	Found     bool
	Value     []byte
	ExpiresAt time.Time
}

type Service interface {
	Get(context.Context, Scope, GetInput) (*Entry, error)
	Set(context.Context, Scope, SetInput) error
	Delete(context.Context, Scope, DeleteInput) error
}

type Runtime struct{ factory *module.Factory[Service] }

func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	f, err := module.NewFactory("cache.service", mode, module.Binding[Service]{Value: local, Available: local != nil}, module.Binding[Service]{Value: delegated, Available: delegated != nil})
	if err != nil {
		return nil, err
	}
	return &Runtime{factory: f}, nil
}
func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.factory.Mode()
}
func (r *Runtime) Cache() (Service, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "cache")
	}
	s, err := r.factory.Resolve()
	if err != nil {
		return nil, err
	}
	return checked{s}, nil
}

type checked struct{ Service }

func validate(scope Scope, key string) error {
	if !hostapi.ValidText(scope.Namespace, 128) || !hostapi.ValidText(key, 512) {
		return ErrInvalidArgument
	}
	id, err := uuid.Parse(scope.TenantUUID)
	if err != nil || id == uuid.Nil || id.String() != scope.TenantUUID || strings.TrimSpace(scope.Namespace) == "" || strings.TrimSpace(scope.Namespace) != scope.Namespace || strings.TrimSpace(key) == "" || strings.TrimSpace(key) != key {
		return ErrInvalidArgument
	}
	return nil
}
func (s checked) Get(ctx context.Context, scope Scope, in GetInput) (*Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validate(scope, in.Key); err != nil {
		return nil, err
	}
	out, err := s.Service.Get(ctx, scope, in)
	if err != nil {
		return nil, err
	}
	if out == nil || (!out.Found && (len(out.Value) != 0 || !out.ExpiresAt.IsZero())) || (out.Found && out.ExpiresAt.IsZero()) {
		return nil, ErrInvalidResponse
	}
	return out, nil
}
func (s checked) Set(ctx context.Context, scope Scope, in SetInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validate(scope, in.Key); err != nil {
		return err
	}
	if in.TTL < time.Millisecond || in.TTL > 24*time.Hour || in.TTL%time.Millisecond != 0 || len(in.Value) > 1<<20 {
		return ErrInvalidArgument
	}
	return s.Service.Set(ctx, scope, in)
}

// Delete is idempotent: an absent key is success, not a dependency failure.
func (s checked) Delete(ctx context.Context, scope Scope, in DeleteInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validate(scope, in.Key); err != nil {
		return err
	}
	return s.Service.Delete(ctx, scope, in)
}
