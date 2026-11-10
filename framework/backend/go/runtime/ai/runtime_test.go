package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type generativeStub struct {
	*powerxai.Client
	text string
}

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
func (s generativeStub) VLMInvoke(context.Context, powerxai.MediaInvokeInput) (*powerxai.AnalysisResult, error) {
	return &powerxai.AnalysisResult{}, nil
}
func (s generativeStub) ImageInvoke(context.Context, powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return &powerxai.MediaTask{}, nil
}
func (s generativeStub) VideoInvoke(context.Context, powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return &powerxai.MediaTask{}, nil
}
func (s generativeStub) TTSInvoke(context.Context, powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return &powerxai.MediaTask{}, nil
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

func TestDelegatedRuntimeLLMStreamUsesCoreGatewayBinding(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/tenant/invocations/stream" || r.Header.Get("Authorization") != "Bearer selected-sts" {
			t.Error("LLM_STREAM_RUNTIME_BYPASSED_GATEWAY")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: end\ndata: {}\n\n")
	}))
	defer server.Close()
	client, err := powerxai.NewClientWithTokenProvider(powerxai.Config{BaseURL: server.URL}, powerxai.TokenProviderFunc(func(context.Context) (string, error) { return "selected-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(provider.ModeDelegated, generativeStub{text: "local"}, client)
	if err != nil {
		t.Fatal(err)
	}
	service, err := runtime.Generative()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.LLMStream(context.Background(), powerxai.LLMStreamInput{LLMInvokeInput: powerxai.LLMInvokeInput{ModelKey: "fixture", Inputs: []powerxai.ContentItem{{Type: "text", Content: "fixture"}}}}, func(powerxai.LLMStreamEvent) error { return nil }); err != nil || calls != 1 {
		t.Fatal("LLM_STREAM_RUNTIME_FAILED", err, calls)
	}
}
