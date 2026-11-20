package host_contract

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"github.com/gin-gonic/gin"
	"time"
)

func decodeStorage(input map[string]any, out any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(out); err != nil {
		return &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return nil
}
func (h *Handler) probeStorage(c *gin.Context, tenant string, input map[string]any, result *hostcontract.ProbeResult) error {
	ctx := c.Request.Context()
	if result.Operation == hostcontract.OperationStatus {
		if len(input) != 0 {
			return &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
		var err error
		if result.Module == hostcontract.ModuleCache {
			_, err = h.cache.Cache()
		} else {
			_, err = h.tasks.Tasks()
		}
		if err == nil {
			result.Result = map[string]any{"adapter_available": true, "connectivity_verified": false}
		}
		return err
	}
	if result.Module == hostcontract.ModuleCache {
		service, err := h.cache.Cache()
		if err != nil {
			return err
		}
		var v struct {
			Namespace string  `json:"namespace"`
			Key       string  `json:"key"`
			Value     *string `json:"value_base64"`
			TTL       *int64  `json:"ttl_ms"`
		}
		if err := decodeStorage(input, &v); err != nil {
			return err
		}
		scope := cache.Scope{TenantUUID: tenant, Namespace: v.Namespace}
		if result.Operation != "set" && (v.Value != nil || v.TTL != nil) {
			return cache.ErrInvalidArgument
		}
		switch result.Operation {
		case "get":
			out, err := service.Get(ctx, scope, cache.GetInput{Key: v.Key})
			if err != nil {
				return err
			}
			var expiry any
			if out.Found {
				expiry = out.ExpiresAt
			}
			result.Result = map[string]any{"found": out.Found, "value_base64": base64.StdEncoding.EncodeToString(out.Value), "expires_at": expiry}
			return nil
		case "set":
			if v.Value == nil || v.TTL == nil || *v.TTL < 1 || *v.TTL > 86400000 {
				return cache.ErrInvalidArgument
			}
			value, err := base64.StdEncoding.Strict().DecodeString(*v.Value)
			if err != nil || base64.StdEncoding.EncodeToString(value) != *v.Value {
				return cache.ErrInvalidArgument
			}
			if err = service.Set(ctx, scope, cache.SetInput{Key: v.Key, Value: value, TTL: time.Duration(*v.TTL) * time.Millisecond}); err != nil {
				return err
			}
		case "delete":
			if err := service.Delete(ctx, scope, cache.DeleteInput{Key: v.Key}); err != nil {
				return err
			}
		}
		result.Result = map[string]any{}
		return nil
	}
	service, err := h.tasks.Tasks()
	if err != nil {
		return err
	}
	scope := taskcenter.Scope{TenantUUID: tenant}
	var out *taskcenter.Task
	switch result.Operation {
	case "create":
		var v struct {
			Type    string          `json:"type"`
			Key     string          `json:"idempotency_key"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := decodeStorage(input, &v); err != nil {
			return err
		}
		out, err = service.Create(ctx, scope, taskcenter.CreateInput{Type: v.Type, IdempotencyKey: v.Key, Payload: v.Payload})
	case "get":
		var v struct {
			UUID string `json:"task_uuid"`
		}
		if err := decodeStorage(input, &v); err != nil {
			return err
		}
		out, err = service.Get(ctx, scope, taskcenter.GetInput{TaskUUID: v.UUID})
	case "update":
		var v struct {
			UUID     string           `json:"task_uuid"`
			Revision uint64           `json:"expected_revision"`
			State    taskcenter.State `json:"state"`
			Progress *int             `json:"progress"`
			Message  string           `json:"message_key"`
			Result   json.RawMessage  `json:"result"`
		}
		if err := decodeStorage(input, &v); err != nil {
			return err
		}
		if v.Progress == nil {
			return taskcenter.ErrInvalidArgument
		}
		out, err = service.Update(ctx, scope, taskcenter.UpdateInput{TaskUUID: v.UUID, ExpectedRevision: v.Revision, State: v.State, Progress: *v.Progress, MessageKey: v.Message, Result: v.Result})
	}
	if err == nil {
		result.Result = map[string]any{"task": out}
	}
	return err
}
