package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

const (
	// HealthzPath 是框架暴露的固定健康检查端点。
	HealthzPath = "/healthz"
	// APIPrefix 是业务路由的统一前缀。
	APIPrefix = "/api/v1"
)

// AttachHTTPServer 构造基础 HTTP 服务器并与 App 绑定。
func AttachHTTPServer(app *bootstrap.App) error {
	if app == nil {
		return errors.New("router: app is nil")
	}

	root := newHTTPRouterRoot()
	app.AttachRouter(&httpRouter{root: root})

	server := &http.Server{
		Addr:    app.Config.Listen,
		Handler: root,
	}

	app.AttachServer(server, func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	})
	return nil
}

// RegisterFrameworkRoutes 注册框架保留端点。
func RegisterFrameworkRoutes(app *bootstrap.App) {
	if app == nil || app.Router == nil {
		return
	}
	app.Router.Handle(http.MethodGet, HealthzPath, func(ctx bootstrap.Context) {
		ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
}

// RegisterPluginRoutes 暴露业务路由挂载点。
func RegisterPluginRoutes(app *bootstrap.App, reg func(rg bootstrap.Router)) {
	if app == nil || app.Router == nil || reg == nil {
		return
	}
	reg(app.Router.Group(APIPrefix))
}

type httpRouterRoot struct {
	mu          sync.RWMutex
	static      map[string]map[string]http.HandlerFunc
	paramRoutes map[string][]routeEntry
	notFound    http.HandlerFunc
}

type routeEntry struct {
	pattern  string
	handler  bootstrap.Handler
	segments []string
}

func newHTTPRouterRoot() *httpRouterRoot {
	return &httpRouterRoot{
		static:      make(map[string]map[string]http.HandlerFunc),
		paramRoutes: make(map[string][]routeEntry),
		notFound: func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
	}
}

func (r *httpRouterRoot) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	methodRoutes := r.static[req.Method]
	handler, ok := methodRoutes[req.URL.Path]
	paramEntries := append([]routeEntry(nil), r.paramRoutes[req.Method]...)
	r.mu.RUnlock()

	if !ok {
		if len(paramEntries) > 0 {
			pathSegments := splitPath(req.URL.Path)
			for _, entry := range paramEntries {
				if params, matched := matchSegments(entry.segments, pathSegments); matched {
					ctx := newHTTPContext(w, req, params)
					entry.handler(ctx)
					return
				}
			}
		}
		r.notFound(w, req)
		return
	}
	handler(w, req)
}

func (r *httpRouterRoot) register(method, fullPath string, handler bootstrap.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if strings.Contains(fullPath, ":") {
		entry := routeEntry{
			pattern:  fullPath,
			handler:  handler,
			segments: splitPath(fullPath),
		}
		r.paramRoutes[method] = append(r.paramRoutes[method], entry)
		return
	}
	if _, ok := r.static[method]; !ok {
		r.static[method] = make(map[string]http.HandlerFunc)
	}
	r.static[method][fullPath] = func(w http.ResponseWriter, req *http.Request) {
		ctx := newHTTPContext(w, req, nil)
		handler(ctx)
	}
}

type httpRouter struct {
	root   *httpRouterRoot
	prefix string
	mws    []bootstrap.Middleware
}

func (r *httpRouter) Group(rel string) bootstrap.Router {
	p := joinPaths(r.prefix, rel)
	return &httpRouter{
		root:   r.root,
		prefix: p,
		mws:    append([]bootstrap.Middleware{}, r.mws...),
	}
}

func (r *httpRouter) Handle(method, p string, h bootstrap.Handler) {
	fullPath := joinPaths(r.prefix, p)
	final := h
	for i := len(r.mws) - 1; i >= 0; i-- {
		final = r.mws[i](final)
	}
	r.root.register(method, fullPath, final)
}

func (r *httpRouter) Use(mw ...bootstrap.Middleware) {
	if len(mw) == 0 {
		return
	}
	r.mws = append(r.mws, mw...)
}

type httpContext struct {
	w   http.ResponseWriter
	req *http.Request
	ctx context.Context

	params map[string]string
}

func (c *httpContext) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

func (c *httpContext) Query(name string) string {
	return c.req.URL.Query().Get(name)
}

func (c *httpContext) BindJSON(v any) error {
	defer c.req.Body.Close()
	return json.NewDecoder(c.req.Body).Decode(v)
}

func (c *httpContext) JSON(code int, v any) {
	c.w.Header().Set("Content-Type", "application/json")
	c.Status(code)
	_ = json.NewEncoder(c.w).Encode(v)
}

func (c *httpContext) Status(code int) {
	c.w.WriteHeader(code)
}

func (c *httpContext) Header(name string) string {
	if v := c.req.Header.Get(name); v != "" {
		return v
	}
	return c.w.Header().Get(name)
}

func (c *httpContext) SetHeader(name, value string) {
	if value == "" {
		c.w.Header().Del(name)
		return
	}
	c.w.Header().Set(name, value)
}

func (c *httpContext) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return c.req.Context()
}

func (c *httpContext) SetContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	c.ctx = ctx
	c.req = c.req.WithContext(ctx)
}

func joinPaths(base, rel string) string {
	if rel == "" || rel == "/" {
		return sanitizePath(base)
	}
	if base == "" || base == "/" {
		return sanitizePath(rel)
	}
	joined := path.Join(base, rel)
	if strings.HasSuffix(rel, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func sanitizePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func newHTTPContext(w http.ResponseWriter, req *http.Request, params map[string]string) *httpContext {
	return &httpContext{
		w:      w,
		req:    req,
		params: params,
		ctx:    req.Context(),
	}
}

func splitPath(p string) []string {
	if p == "" || p == "/" {
		return []string{}
	}
	trimmed := strings.Trim(p, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func matchSegments(patternSegments, pathSegments []string) (map[string]string, bool) {
	if len(patternSegments) != len(pathSegments) {
		return nil, false
	}
	paramValues := make(map[string]string)
	for idx, seg := range patternSegments {
		part := pathSegments[idx]
		if strings.HasPrefix(seg, ":") {
			name := seg[1:]
			paramValues[name] = part
			continue
		}
		if seg != part {
			return nil, false
		}
	}
	return paramValues, true
}
