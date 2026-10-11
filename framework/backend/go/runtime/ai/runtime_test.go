package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type generativeStub struct{ text string }

func (s generativeStub) ListLLMModels(context.Context, string) (*powerxai.ListLLMModelsOutput, error) {
	return &powerxai.ListLLMModelsOutput{Items: []powerxai.LLMModel{{ModelKey: s.text}}}, nil
}

func (s generativeStub) LLMInvoke(context.Context, powerxai.LLMInvokeInput) (*powerxai.LLMInvokeOutput, error) {
	return &powerxai.LLMInvokeOutput{Type: "text", Text: s.text}, nil
}
func (s generativeStub) LLMStream(_ context.Context, _ powerxai.LLMStreamInput, emit func(powerxai.LLMStreamEvent) error) error {
	return emit(powerxai.LLMStreamEvent{Type: "completed", Text: s.text})
}
func (s generativeStub) CreateLLMSession(context.Context, powerxai.CreateLLMSessionInput) (*powerxai.LLMSession, error) {
	return &powerxai.LLMSession{SessionID: s.text}, nil
}
func (s generativeStub) AppendLLMSessionMessage(context.Context, string, powerxai.AppendLLMSessionMessageInput) error {
	return nil
}
func (s generativeStub) LLMSessionStream(_ context.Context, _ string, emit func(powerxai.LLMStreamEvent) error) error {
	return emit(powerxai.LLMStreamEvent{Type: "completed", Text: s.text})
}
func (s generativeStub) EmbeddingInvoke(context.Context, powerxai.EmbeddingInvokeInput) (*powerxai.EmbeddingInvokeOutput, error) {
	return &powerxai.EmbeddingInvokeOutput{}, nil
}
func (s generativeStub) VLMInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error) {
	return &powerxai.ModalInvokeOutput{}, nil
}
func (s generativeStub) ImageInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error) {
	return &powerxai.ModalInvokeOutput{}, nil
}
func (s generativeStub) VideoInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error) {
	return &powerxai.ModalInvokeOutput{}, nil
}
func (s generativeStub) TTSInvoke(context.Context, powerxai.ModalInvokeInput) (*powerxai.ModalInvokeOutput, error) {
	return &powerxai.ModalInvokeOutput{}, nil
}

func TestRuntimeSelectsOnlyConfiguredAdapter(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			runtime, err := NewRuntime(tc.mode, generativeStub{text: "local"}, generativeStub{text: "delegated"})
			if err != nil {
				t.Fatalf("NewRuntime(): %v", err)
			}
			service, err := runtime.Generative()
			if err != nil {
				t.Fatalf("Generative(): %v", err)
			}
			output, err := service.LLMInvoke(context.Background(), powerxai.LLMInvokeInput{})
			if err != nil || output.Text != tc.want {
				t.Fatalf("LLMInvoke() = %#v, %v", output, err)
			}
		})
	}
}

func TestRuntimeFailsClosedForMissingSelectedAdapter(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, generativeStub{text: "local"}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	_, err = runtime.Generative()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Generative() error=%v", err)
	}
}
