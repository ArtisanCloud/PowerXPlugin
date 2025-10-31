package routes

import (
	"net/http"
	"strings"
	"testing"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

type fakeRouter struct {
	prefix string
	routes *[]registeredRoute
}

type registeredRoute struct {
	method  string
	path    string
	handler bootstrap.Handler
}

func (r *fakeRouter) Group(rel string) bootstrap.Router {
	return &fakeRouter{
		prefix: combine(r.prefix, rel),
		routes: r.routes,
	}
}

func (r *fakeRouter) Handle(method, path string, h bootstrap.Handler) {
	full := combine(r.prefix, path)
	*r.routes = append(*r.routes, registeredRoute{
		method:  method,
		path:    full,
		handler: h,
	})
}

func (r *fakeRouter) Use(mw ...bootstrap.Middleware) {}

type fakeContext struct {
	status int
	body   map[string]string
}

func (c *fakeContext) Param(string) string { return "" }
func (c *fakeContext) Query(string) string { return "" }
func (c *fakeContext) BindJSON(any) error  { return nil }
func (c *fakeContext) Status(code int)     { c.status = code }
func (c *fakeContext) JSON(code int, v any) {
	c.status = code
	if payload, ok := v.(map[string]string); ok {
		c.body = payload
	}
}

func TestRegisterAddsPingRoute(t *testing.T) {
	routes := make([]registeredRoute, 0)
	r := &fakeRouter{prefix: "/api/v1", routes: &routes}

	Register(r)

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	route := routes[0]
	if route.method != http.MethodGet {
		t.Fatalf("method = %s, want GET", route.method)
	}
	if route.path != "/api/v1/ping" {
		t.Fatalf("path = %s, want /api/v1/ping", route.path)
	}

	ctx := &fakeContext{}
	route.handler(ctx)
	if ctx.status != http.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.status)
	}
	if ctx.body["status"] != "ok" {
		t.Fatalf("body status = %q, want ok", ctx.body["status"])
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
