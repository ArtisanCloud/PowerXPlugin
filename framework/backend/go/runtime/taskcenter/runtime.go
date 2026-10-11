// Package taskcenter defines task records, not queue delivery or scheduling.
// Host transport uses the published tenant Runtime Host contract.
package taskcenter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/google/uuid"
)

var (
	ErrInvalidArgument = errors.New("TASKCENTER_INVALID_ARGUMENT")
	ErrInvalidResponse = errors.New("TASKCENTER_INVALID_RESPONSE")
	ErrNotFound        = errors.New("TASKCENTER_NOT_FOUND")
	ErrConflict        = errors.New("TASKCENTER_CONFLICT")
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
	Cancelled State = "cancelled"
)

// Scope comes from trusted identity. Storage must isolate tenant + task UUID;
// Core must also isolate the credential's plugin/service actor.
type Scope struct{ TenantUUID string }
type CreateInput struct {
	Type           string
	IdempotencyKey string
	Payload        json.RawMessage
}
type GetInput struct{ TaskUUID string }

// ExpectedRevision provides compare-and-swap updates. Progress is 0..100.
// MessageKey is an i18n key, never a human-readable status message.
type UpdateInput struct {
	TaskUUID         string
	ExpectedRevision uint64
	State            State
	Progress         int
	MessageKey       string
	Result           json.RawMessage
}
type Task struct {
	TaskUUID    string          `json:"task_uuid"`
	TenantUUID  string          `json:"tenant_uuid"`
	Type        string          `json:"type"`
	State       State           `json:"state"`
	Progress    int             `json:"progress"`
	Revision    uint64          `json:"revision"`
	MessageKey  string          `json:"message_key"`
	Result      json.RawMessage `json:"result"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at"`
}
type Service interface {
	Create(context.Context, Scope, CreateInput) (*Task, error)
	Get(context.Context, Scope, GetInput) (*Task, error)
	Update(context.Context, Scope, UpdateInput) (*Task, error)
}
type Runtime struct{ factory *module.Factory[Service] }

func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	f, err := module.NewFactory("taskcenter.service", mode, module.Binding[Service]{Value: local, Available: local != nil}, module.Binding[Service]{Value: delegated, Available: delegated != nil})
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
func (r *Runtime) Tasks() (Service, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "taskcenter")
	}
	s, err := r.factory.Resolve()
	if err != nil {
		return nil, err
	}
	return checked{s}, nil
}
func validUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}
func validState(s State) bool {
	return s == Queued || s == Running || s == Succeeded || s == Failed || s == Cancelled
}
func terminal(s State) bool              { return s == Succeeded || s == Failed || s == Cancelled }
func validJSON(raw json.RawMessage) bool { return hostapi.ValidJSON(raw) }

const MaxRevision uint64 = 1<<63 - 1

type checked struct{ Service }

func validate(ctx context.Context, scope Scope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validUUID(scope.TenantUUID) {
		return ErrInvalidArgument
	}
	return nil
}
func response(scope Scope, id string, out *Task, err error) (*Task, error) {
	if err != nil {
		return nil, err
	}
	if out != nil && out.Revision > MaxRevision {
		return nil, ErrInvalidResponse
	}
	if out == nil || !validUUID(out.TaskUUID) || out.TenantUUID != scope.TenantUUID || (id != "" && out.TaskUUID != id) || !validState(out.State) || out.Revision == 0 || out.Progress < 0 || out.Progress > 100 || (out.State == Succeeded && out.Progress != 100) || (terminal(out.State) && out.CompletedAt == nil) || (!terminal(out.State) && out.CompletedAt != nil) || !validJSON(out.Result) {
		return nil, ErrInvalidResponse
	}
	return out, nil
}
func (s checked) Create(ctx context.Context, scope Scope, in CreateInput) (*Task, error) {
	if err := validate(ctx, scope); err != nil {
		return nil, err
	}
	if !hostapi.ValidKey(in.Type, 128) || !hostapi.ValidText(in.IdempotencyKey, 128) || !validJSON(in.Payload) {
		return nil, ErrInvalidArgument
	}
	out, err := s.Service.Create(ctx, scope, in)
	return response(scope, "", out, err)
}
func (s checked) Get(ctx context.Context, scope Scope, in GetInput) (*Task, error) {
	if err := validate(ctx, scope); err != nil {
		return nil, err
	}
	if !validUUID(in.TaskUUID) {
		return nil, ErrInvalidArgument
	}
	out, err := s.Service.Get(ctx, scope, in)
	return response(scope, in.TaskUUID, out, err)
}
func (s checked) Update(ctx context.Context, scope Scope, in UpdateInput) (*Task, error) {
	if err := validate(ctx, scope); err != nil {
		return nil, err
	}
	if in.ExpectedRevision > MaxRevision || (in.MessageKey != "" && !hostapi.ValidKey(in.MessageKey, 256)) {
		return nil, ErrInvalidArgument
	}
	if !validUUID(in.TaskUUID) || in.ExpectedRevision == 0 || !validState(in.State) || in.Progress < 0 || in.Progress > 100 || (in.State == Succeeded && in.Progress != 100) || !validJSON(in.Result) {
		return nil, ErrInvalidArgument
	}
	out, err := s.Service.Update(ctx, scope, in)
	if err == nil && out != nil && (out.Revision != in.ExpectedRevision+1 || out.State != in.State || out.Progress != in.Progress) {
		return nil, ErrInvalidResponse
	}
	return response(scope, in.TaskUUID, out, err)
}
