package taskcenter

import (
	"context"
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"net/http"
)

const ReadCapability = "com.corex.runtime.taskcenter.read"
const ManageCapability = "com.corex.runtime.taskcenter.manage"

type HostProvider struct{ client *hostapi.Client }

func hostTask(out *Task, err error) (*Task, error) {
	if err != nil {
		return nil, err
	}
	if out.CreatedAt.IsZero() || out.UpdatedAt.IsZero() || out.UpdatedAt.Before(out.CreatedAt) || !hostapi.ValidKey(out.Type, 128) || (out.MessageKey != "" && !hostapi.ValidKey(out.MessageKey, 256)) {
		return nil, ErrInvalidResponse
	}
	return out, nil
}

func NewHostProvider(cfg hostapi.Config, tokens hostapi.TokenProvider, h *http.Client) (Service, error) {
	c, err := hostapi.New(cfg, tokens, h, "TASKCENTER")
	if err != nil {
		return nil, err
	}
	return checked{&HostProvider{c}}, nil
}
func (p *HostProvider) Create(ctx context.Context, scope Scope, in CreateInput) (*Task, error) {
	body := struct {
		Type    string          `json:"type"`
		Key     string          `json:"idempotency_key"`
		Payload json.RawMessage `json:"payload"`
	}{in.Type, in.IdempotencyKey, in.Payload}
	var out Task
	err := p.client.Do(ctx, scope.TenantUUID, "POST", "/api/v1/tenant/runtime/tasks", body, &out)
	return hostTask(&out, err)
}
func (p *HostProvider) Get(ctx context.Context, scope Scope, in GetInput) (*Task, error) {
	var out Task
	err := p.client.Do(ctx, scope.TenantUUID, "GET", "/api/v1/tenant/runtime/tasks/"+in.TaskUUID, nil, &out)
	return hostTask(&out, err)
}
func (p *HostProvider) Update(ctx context.Context, scope Scope, in UpdateInput) (*Task, error) {
	body := struct {
		Revision uint64          `json:"expected_revision"`
		State    State           `json:"state"`
		Progress int             `json:"progress"`
		Message  string          `json:"message_key"`
		Result   json.RawMessage `json:"result"`
	}{in.ExpectedRevision, in.State, in.Progress, in.MessageKey, in.Result}
	var out Task
	err := p.client.Do(ctx, scope.TenantUUID, "PATCH", "/api/v1/tenant/runtime/tasks/"+in.TaskUUID, body, &out)
	return hostTask(&out, err)
}
