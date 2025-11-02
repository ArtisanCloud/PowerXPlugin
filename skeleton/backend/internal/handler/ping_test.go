package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/service"
)

type pingStubContext struct {
	status  int
	payload any
	ctx     context.Context
}

func newPingStub() *pingStubContext {
	return &pingStubContext{ctx: context.Background()}
}

func (p *pingStubContext) Param(string) string            { return "" }
func (p *pingStubContext) Query(string) string            { return "" }
func (p *pingStubContext) BindJSON(any) error             { return nil }
func (p *pingStubContext) Status(code int)                { p.status = code }
func (p *pingStubContext) JSON(code int, v any)           { p.status = code; p.payload = v }
func (p *pingStubContext) Header(string) string           { return "" }
func (p *pingStubContext) SetHeader(string, string)       {}
func (p *pingStubContext) Method() string                 { return http.MethodGet }
func (p *pingStubContext) Context() context.Context       { return p.ctx }
func (p *pingStubContext) SetContext(ctx context.Context) { p.ctx = ctx }

func TestPingHandler(t *testing.T) {
	h := NewPingHandler(service.NewPingService())
	ctx := newPingStub()

	handler := h.Handle()
	handler(ctx)

	if ctx.status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", ctx.status)
	}
	resp, ok := ctx.payload.(map[string]string)
	if !ok {
		t.Fatalf("expected map payload, got %#v", ctx.payload)
	}
	if resp["status"] != "ok" {
		t.Fatalf("unexpected payload: %#v", resp)
	}
}
