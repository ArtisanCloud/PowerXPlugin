package ai

import (
	"context"
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
		body := "event: token\ndata: {\"traceId\":\"trace-1\",\"delta\":\"hi\"}\n\nevent: end\ndata: {\"finish_reason\":\"stop\"}\n\n"
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
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
	if len(events) != 2 || events[0].Delta != "hi" || events[1].FinishReason != "stop" {
		t.Fatalf("events=%+v", events)
	}
	if len(paths) != 2 || paths[0] != "POST /api/v1/ai/llm/stream" || paths[1] != "GET /api/v1/ai/llm/sessions/session-1/stream" {
		t.Fatalf("paths=%v", paths)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
