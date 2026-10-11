package runtimeexample

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
)

func localAITest(t *testing.T, h http.HandlerFunc) *LocalAI {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	s, e := NewLocalAI(&config.LocalAIConfig{Models: []config.LocalAIModel{{Key: "chat", Provider: "ollama", Model: "test", Endpoint: server.URL, Modalities: []string{"llm", "embedding", "vlm"}}}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestLocalAIExecution(t *testing.T) {
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	s := localAITest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected gateway credential")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("invalid JSON")
		}
		if body["model"] != "test" {
			t.Error(body)
		}
		if r.URL.Path == "/api/embed" {
			fmt.Fprint(w, `{"embeddings":[[1,2],[3,4]]}`)
			return
		}
		if r.URL.Path != "/api/chat" {
			t.Error(r.URL.Path)
		}
		if body["stream"] == true {
			fmt.Fprintln(w, `{"message":{"content":"answer"},"done":false}`)
			fmt.Fprintln(w, `{"done":true,"done_reason":"stop"}`)
			return
		}
		fmt.Fprint(w, `{"message":{"content":"answer"},"done":true,"done_reason":"stop","eval_count":2}`)
	})
	models, e := s.ListLLMModels(ctx, "")
	if e != nil || len(models.Items) != 1 {
		t.Fatalf("%+v %v", models, e)
	}
	in := dto.LLMInvokeInput{ModelKey: "chat", Inputs: []dto.ContentItem{{Type: "text", Content: "question"}}}
	out, e := s.LLMInvoke(ctx, in)
	if e != nil || out.Text != "answer" {
		t.Fatalf("%+v %v", out, e)
	}
	var types []string
	e = s.LLMStream(ctx, dto.LLMStreamInput{LLMInvokeInput: in}, func(ev dto.LLMStreamEvent) error { types = append(types, ev.Type); return nil })
	if e != nil || fmt.Sprint(types) != "[delta end]" {
		t.Fatalf("%v %v", types, e)
	}
	vec, e := s.EmbeddingInvoke(ctx, dto.EmbeddingInvokeInput{ModelKey: "chat", Inputs: []string{"a", "b"}})
	if e != nil || len(vec.Vectors) != 2 {
		t.Fatalf("%+v %v", vec, e)
	}
	session, e := s.CreateLLMSession(ctx, dto.CreateLLMSessionInput{ModelKey: "chat"})
	if e != nil {
		t.Fatal(e)
	}
	msg := dto.AppendLLMSessionMessageInput{Role: "user", Content: in.Inputs}
	if e = s.AppendLLMSessionMessage(ctx, session.SessionID, msg); e != nil {
		t.Fatal(e)
	}
	if e = s.LLMSessionStream(authx.ContextWithTenantUUID(context.Background(), other), session.SessionID, func(dto.LLMStreamEvent) error { return nil }); e == nil {
		t.Fatal("cross tenant")
	}
	if e = s.LLMSessionStream(ctx, session.SessionID, func(dto.LLMStreamEvent) error { return nil }); e != nil {
		t.Fatal(e)
	}
	if e = s.LLMSessionStream(ctx, session.SessionID, func(dto.LLMStreamEvent) error { return nil }); e == nil {
		t.Fatal("repeated generation without new input")
	}
}

func TestLocalAIFailsClosed(t *testing.T) {
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	in := dto.LLMInvokeInput{ModelKey: "chat", Inputs: []dto.ContentItem{{Type: "text", Content: "question"}}}
	for _, status := range []int{302, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			s := localAITest(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/other")
				w.WriteHeader(status)
			})
			if _, e := s.LLMInvoke(ctx, in); e == nil {
				t.Fatal(status)
			}
		})
	}
	s := localAITest(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"message":{"content":"partial"},"done":false}`)
	})
	if e := s.LLMStream(ctx, dto.LLMStreamInput{LLMInvokeInput: in}, func(dto.LLMStreamEvent) error { return nil }); e == nil {
		t.Fatal("truncated stream succeeded")
	}
	want := errors.New("callback_failed")
	if e := s.LLMStream(ctx, dto.LLMStreamInput{LLMInvokeInput: in}, func(dto.LLMStreamEvent) error { return want }); !errors.Is(e, want) {
		t.Fatal(e)
	}
	if _, e := s.LLMInvoke(context.Background(), in); e == nil {
		t.Fatal("missing tenant")
	}
	if _, e := s.ImageInvoke(ctx, dto.ModalInvokeInput{ModelKey: "chat"}); e == nil {
		t.Fatal("unsupported modality")
	}
	empty, _ := NewLocalAI(nil, nil)
	_, e := empty.LLMInvoke(ctx, in)
	var contract *hostapi.HTTPError
	if !errors.As(e, &contract) || contract.ReasonCode != "AI_MODEL_NOT_CONFIGURED" {
		t.Fatal(e)
	}
	in.Params = map[string]any{"endpoint": "http://evil"}
	if _, e := s.LLMInvoke(ctx, in); e == nil {
		t.Fatal("request endpoint override")
	}
}
