// Package ai defines the mode-bound Framework contract for generative AI
// invocations. Business services depend on this contract instead of selecting
// a local SDK or constructing PowerX HTTP requests.
package ai

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// GenerativeService is the full currently-published AI invocation contract.
// Local adapters are plugin-owned; delegated adapters use the typed PowerX
// client. The Factory selects one adapter at bootstrap and never falls back.
type GenerativeService interface {
	ListLLMModels(context.Context, string) (*powerxai.ListLLMModelsOutput, error)
	LLMInvoke(context.Context, powerxai.LLMInvokeInput) (*powerxai.LLMInvokeOutput, error)
	LLMStream(context.Context, powerxai.LLMStreamInput, func(powerxai.LLMStreamEvent) error) error
	CreateLLMSession(context.Context, powerxai.CreateLLMSessionInput) (*powerxai.LLMSession, error)
	AppendLLMSessionMessage(context.Context, string, powerxai.AppendLLMSessionMessageInput) error
	LLMSessionStream(context.Context, string, func(powerxai.LLMStreamEvent) error) error
	EmbeddingInvoke(context.Context, powerxai.EmbeddingInvokeInput) (*powerxai.EmbeddingInvokeOutput, error)
	VLMInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error)
	ImageInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error)
	VideoInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error)
	TTSInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error)
}

type Runtime struct {
	mode       provider.Mode
	generative *module.Factory[GenerativeService]
}

// NewRuntime binds exactly one supplied adapter to the trusted startup mode.
func NewRuntime(mode provider.Mode, local, delegated GenerativeService) (*Runtime, error) {
	generative, err := module.NewFactory("ai.generative", mode,
		module.Binding[GenerativeService]{Value: local, Available: local != nil},
		module.Binding[GenerativeService]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, generative: generative}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

func (r *Runtime) Generative() (GenerativeService, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "ai runtime is unavailable")
	}
	return r.generative.Resolve()
}
