// Package notifications defines the Framework-owned dual-mode notification
// publishing boundary.
package notifications

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Publisher creates tenant-scoped notifications. The target member UUID is an
// optional delivery target, not a tenant selector.
type Publisher interface {
	Create(context.Context, powerxnotifications.CreateInput) (*powerxnotifications.Notification, error)
}

type Runtime struct {
	mode      provider.Mode
	channel   string
	publisher *module.Factory[Publisher]
}

func NewRuntime(mode provider.Mode, local, delegated Publisher) (*Runtime, error) {
	publisher, err := module.NewFactory("notifications.publisher", mode,
		module.Binding[Publisher]{Value: local, Available: local != nil},
		module.Binding[Publisher]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, channel: string(mode), publisher: publisher}, nil
}

// NewCoreHostRuntime explicitly selects API-key Core simulation at startup.
// It is separate from the pure local database adapter and never fails over.
func NewCoreHostRuntime(client *powerxnotifications.Client) (*Runtime, error) {
	if client == nil || client.CredentialScheme() != "apikey" {
		return nil, module.NewError("FRAMEWORK_NOTIFICATION_CORE_HOST_UNAVAILABLE", "FRAMEWORK_NOTIFICATION_CORE_HOST_UNAVAILABLE")
	}
	runtime, err := NewRuntime(provider.ModeLocal, client, nil)
	if err == nil {
		runtime.channel = "core_host"
	}
	return runtime, err
}
func (r *Runtime) Channel() string {
	if r == nil {
		return ""
	}
	return r.channel
}

func (r *Runtime) CardInvalidator() (powerxnotifications.CardInvalidator, error) {
	publisher, err := r.Publisher()
	if err != nil {
		return nil, err
	}
	invalidator, ok := publisher.(powerxnotifications.CardInvalidator)
	if !ok {
		return nil, module.NewError("FRAMEWORK_NOTIFICATION_CARD_ADAPTER_UNAVAILABLE", "FRAMEWORK_NOTIFICATION_CARD_ADAPTER_UNAVAILABLE")
	}
	return invalidator, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

func (r *Runtime) Publisher() (Publisher, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "notifications runtime is unavailable")
	}
	return r.publisher.Resolve()
}

// ReadAcknowledger resolves the same selected adapter; it never switches mode.
func (r *Runtime) ReadAcknowledger() (powerxnotifications.ReadAcknowledger, error) {
	publisher, err := r.Publisher()
	if err != nil {
		return nil, err
	}
	reader, ok := publisher.(powerxnotifications.ReadAcknowledger)
	if !ok {
		return nil, module.NewError("FRAMEWORK_NOTIFICATION_READ_ADAPTER_UNAVAILABLE", "FRAMEWORK_NOTIFICATION_READ_ADAPTER_UNAVAILABLE")
	}
	return reader, nil
}
