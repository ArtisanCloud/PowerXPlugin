package routes

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	"github.com/powerx-plugin/framework/backend/go/router"
)

type registeredRoute struct {
	method  string
	path    string
	handler bootstrap.Handler
}

type fakeRouter struct {
	prefix string
	routes *[]registeredRoute
	mws    []bootstrap.Middleware
}

func (r *fakeRouter) Group(rel string) bootstrap.Router {
	return &fakeRouter{
		prefix: combine(r.prefix, rel),
		routes: r.routes,
		mws:    append([]bootstrap.Middleware{}, r.mws...),
	}
}

func (r *fakeRouter) Handle(method, path string, h bootstrap.Handler) {
	full := combine(r.prefix, path)
	final := h
	for i := len(r.mws) - 1; i >= 0; i-- {
		final = r.mws[i](final)
	}
	*r.routes = append(*r.routes, registeredRoute{method: method, path: full, handler: final})
}

func (r *fakeRouter) Use(mw ...bootstrap.Middleware) {
	if len(mw) == 0 {
		return
	}
	r.mws = append(r.mws, mw...)
}

type fakeContext struct {
	headers map[string]string
	params  map[string]string
	status  int
	payload any
	ctx     context.Context
	method  string
}

func newFakeContext(headers map[string]string, params map[string]string) *fakeContext {
	return &fakeContext{
		headers: headers,
		params:  params,
		method:  http.MethodGet,
	}
}

func (c *fakeContext) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

func (c *fakeContext) Query(string) string { return "" }
func (c *fakeContext) BindJSON(any) error  { return nil }
func (c *fakeContext) Status(code int)     { c.status = code }
func (c *fakeContext) JSON(code int, v any) {
	c.status = code
	c.payload = v
}
func (c *fakeContext) Header(name string) string {
	if c.headers == nil {
		return ""
	}
	return c.headers[name]
}
func (c *fakeContext) SetHeader(name, value string) {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[name] = value
}
func (c *fakeContext) Method() string {
	if c.method == "" {
		return http.MethodGet
	}
	return c.method
}
func (c *fakeContext) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}
func (c *fakeContext) SetContext(ctx context.Context) { c.ctx = ctx }

func TestRegisterRegistersAllRoutes(t *testing.T) {
	routes := make([]registeredRoute, 0)
	r := &fakeRouter{prefix: "/api/v1", routes: &routes}

	Register(r)

	if len(routes) != 6 {
		t.Fatalf("expected 6 routes, got %d", len(routes))
	}

	// 验证 ping 路由输出
	var ping registeredRoute
	for _, rt := range routes {
		if rt.method == http.MethodGet && rt.path == "/api/v1/ping" {
			ping = rt
			break
		}
	}
	ctx := newFakeContext(nil, nil)
	ping.handler(ctx)
	if ctx.status != http.StatusOK {
		t.Fatalf("ping status = %d, want 200", ctx.status)
	}
	switch payload := ctx.payload.(type) {
	case router.Envelope:
		if !payload.Success {
			t.Fatalf("unexpected envelope: %#v", payload)
		}
	case map[string]string:
		if payload["status"] != "ok" {
			t.Fatalf("unexpected ping payload: %#v", payload)
		}
	default:
		t.Fatalf("unexpected ping payload type: %#v", ctx.payload)
	}

	// 验证模板列表路由可以执行并返回成功
	var list registeredRoute
	for _, rt := range routes {
		if rt.method == http.MethodGet && rt.path == "/api/v1/templates" {
			list = rt
			break
		}
	}
	listCtx := newFakeContext(map[string]string{"X-Tenant-ID": "1"}, nil)
	list.handler(listCtx)
	if listCtx.status != http.StatusOK {
		t.Fatalf("templates list status = %d, want 200", listCtx.status)
	}
	if env, ok := listCtx.payload.(router.Envelope); !ok || env.Success != true {
		t.Fatalf("unexpected templates payload: %#v", listCtx.payload)
	}
}

func combine(prefix, rel string) string {
	p := strings.TrimSuffix(prefix, "/")
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "/" {
		if p == "" {
			return "/"
		}
		return ensureLeadingSlash(p)
	}
	if strings.HasPrefix(rel, "/") {
		rel = rel[1:]
	}
	if p == "" || p == "/" {
		return ensureLeadingSlash(rel)
	}
	return ensureLeadingSlash(p + "/" + rel)
}

func ensureLeadingSlash(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}
