// Package runtimeexample provides process-local Skeleton adapters, not production storage.
// Restarting Skeleton clears these records. Delegated mode never constructs them.
package runtimeexample

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"github.com/google/uuid"
)

type cacheKey struct{ tenant, namespace, key string }
type MemoryCache struct {
	mu      sync.Mutex
	entries map[cacheKey]cache.Entry
	now     func() time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: map[cacheKey]cache.Entry{}, now: time.Now}
}
func (s *MemoryCache) Get(ctx context.Context, scope cache.Scope, in cache.GetInput) (*cache.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := cacheKey{scope.TenantUUID, scope.Namespace, in.Key}
	value, ok := s.entries[key]
	if !ok || !s.now().Before(value.ExpiresAt) {
		delete(s.entries, key)
		return &cache.Entry{}, nil
	}
	value.Value = append([]byte(nil), value.Value...)
	return &value, nil
}
func (s *MemoryCache) Set(ctx context.Context, scope cache.Scope, in cache.SetInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[cacheKey{scope.TenantUUID, scope.Namespace, in.Key}] = cache.Entry{Found: true, Value: append([]byte(nil), in.Value...), ExpiresAt: s.now().Add(in.TTL)}
	return nil
}
func (s *MemoryCache) Delete(ctx context.Context, scope cache.Scope, in cache.DeleteInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, cacheKey{scope.TenantUUID, scope.Namespace, in.Key})
	return nil
}

type taskKey struct{ tenant, key string }
type record struct {
	task    taskcenter.Task
	payload any
}
type MemoryTasks struct {
	mu      sync.Mutex
	records map[string]*record
	keys    map[taskKey]string
}

func NewMemoryTasks() *MemoryTasks {
	return &MemoryTasks{records: map[string]*record{}, keys: map[taskKey]string{}}
}
func clone(t taskcenter.Task) *taskcenter.Task {
	t.Result = append(json.RawMessage(nil), t.Result...)
	if t.CompletedAt != nil {
		v := *t.CompletedAt
		t.CompletedAt = &v
	}
	return &t
}
func (s *MemoryTasks) Create(ctx context.Context, scope taskcenter.Scope, in taskcenter.CreateInput) (*taskcenter.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var payload any
	if len(in.Payload) > 0 {
		d := json.NewDecoder(bytes.NewReader(in.Payload))
		d.UseNumber()
		if err := d.Decode(&payload); err != nil {
			return nil, taskcenter.ErrInvalidArgument
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := taskKey{scope.TenantUUID, in.IdempotencyKey}
	if id, ok := s.keys[key]; ok {
		r := s.records[id]
		if r.task.Type != in.Type || !reflect.DeepEqual(r.payload, payload) {
			return nil, taskcenter.ErrConflict
		}
		return clone(r.task), nil
	}
	now := time.Now().UTC()
	t := taskcenter.Task{TaskUUID: uuid.NewString(), TenantUUID: scope.TenantUUID, Type: in.Type, State: taskcenter.Queued, Revision: 1, Result: json.RawMessage("null"), CreatedAt: now, UpdatedAt: now}
	s.records[t.TaskUUID] = &record{t, payload}
	s.keys[key] = t.TaskUUID
	return clone(t), nil
}
func (s *MemoryTasks) Get(ctx context.Context, scope taskcenter.Scope, in taskcenter.GetInput) (*taskcenter.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.records[in.TaskUUID]
	if r == nil || r.task.TenantUUID != scope.TenantUUID {
		return nil, taskcenter.ErrNotFound
	}
	return clone(r.task), nil
}
func (s *MemoryTasks) Update(ctx context.Context, scope taskcenter.Scope, in taskcenter.UpdateInput) (*taskcenter.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.records[in.TaskUUID]
	if r == nil {
		return nil, taskcenter.ErrNotFound
	}
	if err := taskcenter.ValidateTransition(scope, &r.task, in); err != nil {
		return nil, err
	}
	r.task.State = in.State
	r.task.Progress = in.Progress
	r.task.Revision++
	r.task.MessageKey = in.MessageKey
	r.task.Result = append(json.RawMessage(nil), in.Result...)
	if len(r.task.Result) == 0 {
		r.task.Result = json.RawMessage("null")
	}
	r.task.UpdatedAt = time.Now().UTC()
	if in.State == taskcenter.Succeeded || in.State == taskcenter.Failed || in.State == taskcenter.Cancelled {
		now := r.task.UpdatedAt
		r.task.CompletedAt = &now
	}
	return clone(r.task), nil
}
