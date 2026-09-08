// Package module provides the common startup-time selection boundary for
// Framework business modules. It deliberately selects only a supplied adapter;
// it never constructs a fallback adapter or changes mode at request time.
package module

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Binding is one explicitly supplied adapter. Available must be true even when
// Value's zero value is meaningful; this keeps missing adapters fail-closed.
type Binding[T any] struct {
	Value     T
	Available bool
}

// Factory is a mode-bound adapter selector for one named business module.
type Factory[T any] struct {
	module    string
	mode      provider.Mode
	local     Binding[T]
	delegated Binding[T]
}

func NewFactory[T any](module string, mode provider.Mode, local, delegated Binding[T]) (*Factory[T], error) {
	if strings.TrimSpace(module) == "" {
		return nil, NewError("FRAMEWORK_MODULE_INVALID", "module is required")
	}
	if mode != provider.ModeLocal && mode != provider.ModeDelegated {
		return nil, NewError("FRAMEWORK_MODULE_MODE_INVALID", "provider mode is invalid")
	}
	return &Factory[T]{module: strings.TrimSpace(module), mode: mode, local: local, delegated: delegated}, nil
}

func (f *Factory[T]) Mode() provider.Mode {
	if f == nil {
		return ""
	}
	return f.mode
}

// Resolve returns only the adapter for the factory's startup mode.
func (f *Factory[T]) Resolve() (T, error) {
	var zero T
	if f == nil {
		return zero, NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "module factory is unavailable")
	}
	binding := f.local
	if f.mode == provider.ModeDelegated {
		binding = f.delegated
	}
	value := reflect.ValueOf(binding.Value)
	isNil := !value.IsValid()
	if value.IsValid() {
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			isNil = value.IsNil()
		}
	}
	if !binding.Available || isNil {
		return zero, NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", fmt.Sprintf("%s adapter is unavailable for %s", f.module, f.mode))
	}
	return binding.Value, nil
}

type Error struct{ Code, Message string }

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}
func NewError(code, message string) *Error { return &Error{Code: code, Message: message} }
