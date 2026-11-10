package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLLMStreamsUseSSEAndPreserveEvents(t *testing.T) {
	var paths []string
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "sts-token"}, &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.Method+" "+req.URL.RequestURI())
		if req.Header.Get("Authorization") != "Bearer sts-token" || req.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("unexpected headers: %v", req.Header)
		}
		if req.Method == http.MethodPost {
			var input map[string]any
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			payload := input["payload"].(map[string]any)
			if input["capability_id"] != "com.corex.ai.llm.stream" || input["preferred_protocol"] != "core_internal" || len(payload) != 1 || payload["body"].(map[string]any)["model_key"] != "m" {
				t.Fatal("LLM_STREAM_BINDING_INVALID")
			}
		}
		body := "event: delta\ndata: {\"traceId\":\"trace-1\",\"delta\":\"hi\"}\n\nevent: done\ndata: {\"ok\":true,\"finish_reason\":\"stop\"}\n\n"
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	var events []LLMStreamEvent
	if err := client.LLMStream(context.Background(), LLMStreamInput{LLMInvokeInput: LLMInvokeInput{ModelKey: "m", Inputs: []ContentItem{{Role: "user", Type: "text", Content: "hello"}}}}, func(event LLMStreamEvent) error { events = append(events, event); return nil }); err != nil {
		t.Fatal(err)
	}
	if err := client.LLMSessionStream(context.Background(), "session-1", func(event LLMStreamEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Type != "delta" || events[0].Delta != "hi" || events[1].Type != "final" || events[2].Type != "end" || events[2].FinishReason != "stop" {
		t.Fatalf("events=%+v", events)
	}
	if len(paths) != 2 || paths[0] != "POST /api/v1/tenant/invocations/stream" || paths[1] != "GET /api/v1/ai/llm/sessions/session-1/stream" {
		t.Fatalf("paths=%v", paths)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestLLMStreamRejectsUnterminatedAndNonContractEvents(t *testing.T) {
	for _, body := range []string{
		"event: delta\ndata: {\"delta\":\"fixture\"}\n\n",
		"event: end\ndata: {}\n\n",
		"event: completed\ndata: {}\n\n",
		"event: done\ndata: {\"ok\":false}\n\n",
		"event: done\ndata: {\"ok\":true}\n\nevent: delta\ndata: {\"delta\":\"late\"}\n\n",
	} {
		client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "fixture-token"}, &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		})})
		if err != nil {
			t.Fatal(err)
		}
		err = client.LLMStream(context.Background(), LLMStreamInput{LLMInvokeInput: LLMInvokeInput{ModelKey: "fixture", Inputs: []ContentItem{{Type: "text", Content: "fixture"}}}}, func(LLMStreamEvent) error { return nil })
		if err == nil {
			t.Fatal("LLM_STREAM_INVALID_COMPLETION_ACCEPTED")
		}
	}
}

func TestLLMStreamPreservesCoreAuthorizationReason(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "fixture-token"}, &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error_code":"CAPABILITY_FORBIDDEN","reason_code":"CAPABILITY_FORBIDDEN","trace_id":"b3d0388d-b6eb-4500-a98d-22f4eb60c277"}`)), Request: req}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	err = client.LLMStream(context.Background(), LLMStreamInput{LLMInvokeInput: LLMInvokeInput{ModelKey: "fixture", Inputs: []ContentItem{{Type: "text", Content: "fixture"}}}}, func(LLMStreamEvent) error { return nil })
	typed, ok := err.(*HTTPError)
	if !ok || typed.StatusCode != 403 || typed.ReasonCode != "CAPABILITY_FORBIDDEN" || typed.TraceID != "b3d0388d-b6eb-4500-a98d-22f4eb60c277" {
		t.Fatalf("LLM_STREAM_AUTH_DIAGNOSTIC_LOST: %T", err)
	}
}
