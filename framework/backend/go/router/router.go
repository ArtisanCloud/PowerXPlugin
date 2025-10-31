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
	mu       sync.RWMutex
	routes   map[string]map[string]http.HandlerFunc
	notFound http.HandlerFunc
}

func newHTTPRouterRoot() *httpRouterRoot {
	return &httpRouterRoot{
		routes: make(map[string]map[string]http.HandlerFunc),
		notFound: func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
	}
}

func (r *httpRouterRoot) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	methodRoutes := r.routes[req.Method]
	handler, ok := methodRoutes[req.URL.Path]
	r.mu.RUnlock()

	if !ok {
		r.notFound(w, req)
		return
	}
	handler(w, req)
}

func (r *httpRouterRoot) register(method, fullPath string, handler http.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = make(map[string]http.HandlerFunc)
	}
	r.routes[method][fullPath] = handler
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

	r.root.register(method, fullPath, func(w http.ResponseWriter, req *http.Request) {
		ctx := &httpContext{w: w, req: req}
		final(ctx)
	})
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
}

func (c *httpContext) Param(name string) string {
	// 当前实现不解析 Path 参数，占位保持接口稳定。
	return ""
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
