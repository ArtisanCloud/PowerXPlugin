// Package runtimeidentity implements the fixed Core runtime identity service contract.
package runtimeidentity

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

const CapabilityID = "com.corex.runtime.identity.read"

type Identity struct {
	RuntimeMode         string `json:"runtime_mode"`
	CoreVersion         string `json:"core_version"`
	DeploymentEnv       string `json:"deployment_env"`
	PluginID            string `json:"plugin_id"`
	RuntimePluginID     string `json:"runtime_plugin_id"`
	PluginVersion       string `json:"plugin_version"`
	PluginVersionSource string `json:"plugin_version_source"`
	PluginState         string `json:"plugin_state"`
	TraceID             string `json:"trace_id,omitempty"`
	RequestID           string `json:"request_id,omitempty"`
}

type Invoker interface {
	Invoke(context.Context, capability.InvokeInput) (*capability.InvokeResult, error)
}
type Client struct{ invoker Invoker }
type Error struct {
	StatusCode                     int
	ReasonCode, TraceID, RequestID string
}

func (e *Error) Error() string          { return e.ReasonCode }
func NewClient(invoker Invoker) *Client { return &Client{invoker: invoker} }

func (c *Client) ReadRuntimeIdentity(ctx context.Context, pluginID string) (*Identity, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`).MatchString(pluginID) {
		return nil, &Error{StatusCode: 400, ReasonCode: "RUNTIME_IDENTITY_INVALID_ARGUMENT"}
	}
	if c == nil || c.invoker == nil {
		return nil, &Error{StatusCode: 503, ReasonCode: "RUNTIME_IDENTITY_UNAVAILABLE"}
	}
	result, err := c.invoker.Invoke(ctx, capability.InvokeInput{CapabilityID: CapabilityID, PreferredProtocol: "core_internal", Payload: map[string]any{
		"method": "INVOKE", "endpoint": "core://runtime/identity", "body": map[string]any{"operation": "get", "plugin_id": pluginID},
	}})
	if err != nil {
		var remote *capability.HTTPError
		if errors.As(err, &remote) {
			return nil, &Error{StatusCode: remote.StatusCode, ReasonCode: remote.ReasonCode, TraceID: remote.TraceID, RequestID: remote.RequestID}
		}
		return nil, &Error{StatusCode: 503, ReasonCode: "RUNTIME_IDENTITY_UNAVAILABLE"}
	}
	if result == nil || result.FallbackUsed {
		return nil, &Error{StatusCode: 502, ReasonCode: "RUNTIME_IDENTITY_CONTRACT_INVALID"}
	}
	raw, err := json.Marshal(result.Payload["item"])
	var item Identity
	if err != nil || json.Unmarshal(raw, &item) != nil || item.RuntimeMode != "powerx" || item.PluginID != pluginID || item.RuntimePluginID != pluginID || item.PluginVersionSource != "registry" || item.CoreVersion == "" || item.PluginVersion == "" || !validEnv(item.DeploymentEnv) || !validState(item.PluginState) {
		return nil, &Error{StatusCode: 502, ReasonCode: "RUNTIME_IDENTITY_CONTRACT_INVALID", TraceID: result.TraceID}
	}
	item.TraceID = result.TraceID
	return &item, nil
}

func validEnv(value string) bool {
	switch value {
	case "dev", "test", "staging", "prod":
		return true
	}
	return false
}
func validState(value string) bool {
	switch value {
	case "starting", "running", "unhealthy", "stopped", "exited":
		return true
	}
	return false
}
