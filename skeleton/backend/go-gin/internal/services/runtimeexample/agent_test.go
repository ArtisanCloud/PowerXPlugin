package runtimeexample

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/google/uuid"
)

func newAgentTest(t *testing.T, h http.HandlerFunc) (*LocalAgent, string) {
	t.Helper()
	ai := localAITest(t, h)
	id := uuid.NewString()
	s, e := NewLocalAgent("test.plugin", &config.LocalAIConfig{Agents: []config.LocalAgent{{UUID: id, Name: "test", ModelKey: "chat"}}}, ai)
	if e != nil {
		t.Fatal(e)
	}
	return s, id
}
func TestLocalAgentSessionLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(authx.ContextWithTenantUUID(context.Background(), tenant), 5*time.Second)
	defer cancel()
	s, agent := newAgentTest(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"done":true,"message":{"content":"answer"}}`)
	})
	v, e := s.CreateSession(ctx, dto.CreateSessionInput{AgentUUID: agent, Title: "test"})
	if e != nil {
		t.Fatal(e)
	}
	msg := dto.AppendSessionMessageInput{Role: "user", Content: "question"}
	m, e := s.AppendSessionMessage(ctx, v.SessionUUID, "append", msg)
	if e != nil {
		t.Fatal(e)
	}
	m2, e := s.AppendSessionMessage(ctx, v.SessionUUID, "append", msg)
	if e != nil || m.MessageUUID != m2.MessageUUID {
		t.Fatal(e)
	}
	if _, e = s.AppendSessionMessage(ctx, v.SessionUUID, "append", dto.AppendSessionMessageInput{Role: "user", Content: "different"}); e == nil {
		t.Fatal("idempotency collision")
	}
	if _, e = s.AppendSessionMessage(ctx, v.SessionUUID, "system", dto.AppendSessionMessageInput{Role: "system", Content: "inject"}); e == nil {
		t.Fatal("role injection")
	}
	r, e := s.InvokeSession(ctx, v.SessionUUID, m.MessageUUID, "invoke")
	if e != nil {
		t.Fatal(e)
	}
	r2, e := s.InvokeSession(ctx, v.SessionUUID, m.MessageUUID, "invoke")
	if e != nil || r.InvocationUUID != r2.InvocationUUID {
		t.Fatal(e)
	}
	var events []string
	if e = s.StreamSessionEvents(ctx, v.SessionUUID, r.InvocationUUID, func(ev dto.SessionEvent) error { events = append(events, ev.Type); return nil }); e != nil {
		t.Fatal(e)
	}
	if fmt.Sprint(events) != "[state final end]" {
		t.Fatal(events)
	}
	out, e := s.GetSessionInvocation(ctx, v.SessionUUID, r.InvocationUUID)
	if e != nil || out.Status != "succeeded" || out.Output != "answer" {
		t.Fatalf("%+v %v", out, e)
	}
	history, e := s.ListSessionMessages(ctx, v.SessionUUID, dto.SessionPage{})
	if e != nil || len(history.Items) != 2 || history.Items[1].Role != "assistant" {
		t.Fatalf("%+v %v", history, e)
	}
	cross := authx.ContextWithTenantUUID(context.Background(), other)
	if _, e = s.GetSession(cross, v.SessionUUID); e == nil {
		t.Fatal("cross tenant")
	}
	if _, e = s.GetSessionInvocation(cross, v.SessionUUID, r.InvocationUUID); e == nil {
		t.Fatal("cross tenant run")
	}
	if _, e = s.Freeze(ctx, agent, dto.BridgeControlInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateSession(ctx, dto.CreateSessionInput{AgentUUID: agent, Title: "blocked"}); e == nil {
		t.Fatal("frozen")
	}
	if _, e = s.Recover(ctx, agent, dto.BridgeControlInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ArchiveSession(ctx, v.SessionUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AppendSessionMessage(ctx, v.SessionUUID, "next", msg); e == nil {
		t.Fatal("archived")
	}
	if _, e = s.DeleteSession(ctx, v.SessionUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GetSession(ctx, v.SessionUUID); e == nil {
		t.Fatal("deleted")
	}
}

func TestLocalAgentCancelAndDisconnect(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	release := make(chan struct{})
	defer close(release)
	stopped := make(chan struct{})
	s, id := newAgentTest(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		once.Do(func() { close(started) })
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-release:
		}
	})
	ctx, cancel := context.WithTimeout(authx.ContextWithTenantUUID(context.Background(), tenant), 5*time.Second)
	defer cancel()
	v, e := s.CreateSession(ctx, dto.CreateSessionInput{AgentUUID: id, Title: "test"})
	if e != nil {
		t.Fatal(e)
	}
	m, e := s.AppendSessionMessage(ctx, v.SessionUUID, "m", dto.AppendSessionMessageInput{Role: "user", Content: "q"})
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.InvokeSession(ctx, v.SessionUUID, m.MessageUUID, "run")
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	disconnected, stop := context.WithCancel(ctx)
	e = s.StreamSessionEvents(disconnected, v.SessionUUID, r.InvocationUUID, func(dto.SessionEvent) error { stop(); return nil })
	if e == nil {
		t.Fatal("disconnect")
	}
	current, e := s.GetSessionInvocation(ctx, v.SessionUUID, r.InvocationUUID)
	if e != nil || current.Status != "running" {
		t.Fatalf("disconnect cancelled execution: %+v %v", current, e)
	}
	current, e = s.CancelSessionInvocation(ctx, v.SessionUUID, r.InvocationUUID)
	if e != nil || current.Status != "cancelled" {
		t.Fatalf("%+v %v", current, e)
	}
	select {
	case <-stopped:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var events []string
	e = s.StreamSessionEvents(ctx, v.SessionUUID, r.InvocationUUID, func(ev dto.SessionEvent) error { events = append(events, ev.Type); return nil })
	if e == nil || fmt.Sprint(events) != "[state error end]" {
		t.Fatalf("%v %v", events, e)
	}
}
