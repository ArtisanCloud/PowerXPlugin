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
	return &Runtime{mode: mode, publisher: publisher}, nil
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
