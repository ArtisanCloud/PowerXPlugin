package devapi_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/handlers"
)

type stubContext struct {
	body    []byte
	status  int
	payload any
	headers map[string]string
	ctx     context.Context
}

func newStubContext(body []byte) *stubContext {
	return &stubContext{body: body, headers: make(map[string]string), ctx: context.Background()}
}

func (s *stubContext) Param(string) string            { return "" }
func (s *stubContext) Query(string) string            { return "" }
func (s *stubContext) BindJSON(v any) error           { return json.Unmarshal(s.body, v) }
func (s *stubContext) JSON(code int, v any)           { s.status = code; s.payload = v }
func (s *stubContext) Status(code int)                { s.status = code }
func (s *stubContext) Header(name string) string      { return s.headers[name] }
func (s *stubContext) SetHeader(name, value string)   { s.headers[name] = value }
func (s *stubContext) Method() string                 { return "POST" }
func (s *stubContext) Context() context.Context       { return s.ctx }
func (s *stubContext) SetContext(ctx context.Context) { s.ctx = ctx }

func TestRegisterReturnsCreated(t *testing.T) {
	h := handlers.NewDevPluginHandler(nil)
	body, _ := json.Marshal(handlers.RegisterRequest{Manifest: handlers.Manifest{ID: "demo", Version: "1.0.0"}})
	ctx := newStubContext(body)

	h.Register(ctx)

	if ctx.status != 201 {
		t.Fatalf("expected 201, got %d", ctx.status)
	}
	envelope, ok := ctx.payload.(router.Envelope)
	if !ok || !envelope.Success {
		t.Fatalf("expected router envelope payload")
	}
	resp, ok := envelope.Data.(handlers.RegisterResponse)
	if !ok || resp.SessionID == "" || resp.ReloadToken == "" {
		t.Fatalf("expected register response data, got %v", envelope.Data)
	}
}
