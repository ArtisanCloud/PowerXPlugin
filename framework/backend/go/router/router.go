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

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/services/capability_invoker"
)

const (
	// HealthzPath 是框架暴露的固定健康检查端点。
	HealthzPath = "/healthz"
	// APIPrefix 是业务路由的统一前缀。
	APIPrefix = "/api/v1"
	// contractStatusHeader 将契约信息传递给前端。
	contractStatusHeader = "X-PowerX-Contract-Status"
)

var capabilityProxyForwardHeaders = []string{
	"X-PX-Use-Mock",
}

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
	registerCapabilityProxy(app)
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
	if strings.Contains(fullPath, ":") || strings.Contains(fullPath, "*") {
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

func (c *httpContext) Method() string {
	if c.req == nil {
		return ""
	}
	return c.req.Method
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

func (c *httpContext) HTTPResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *httpContext) HTTPRequest() *http.Request {
	return c.req
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
	if len(patternSegments) > len(pathSegments) {
		return nil, false
	}

	paramValues := make(map[string]string)
	for idx := 0; idx < len(patternSegments); idx++ {
		seg := patternSegments[idx]

		if strings.HasPrefix(seg, "*") {
			name := seg[1:]
			paramValues[name] = strings.Join(pathSegments[idx:], "/")
			return paramValues, true
		}

		if idx >= len(pathSegments) {
			return nil, false
		}

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

	if len(patternSegments) != len(pathSegments) {
		return nil, false
	}

	return paramValues, true
}

func registerCapabilityProxy(app *bootstrap.App) {
	if app == nil || app.Router == nil {
		return
	}
	client := app.GatewayClient()
	if client == nil {
		return
	}
	service := capabilityinvoker.NewService(client, app.Logger)
	statusProvider := func() *gateway.ContractStatus {
		if client == nil {
			return nil
		}
		return client.ContractStatus()
	}
	group := app.Router.Group(APIPrefix)
	group.Handle(http.MethodPost, "/integration/capabilities/invoke", capabilityInvokeHandler(service, statusProvider))
}

type capabilityInvokeRequest struct {
	CapabilityID string                 `json:"capabilityId"`
	Action       string                 `json:"action"`
	Payload      map[string]any         `json:"payload"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func capabilityInvokeHandler(service *capabilityinvoker.Service, statusProvider func() *gateway.ContractStatus) bootstrap.Handler {
	return func(ctx bootstrap.Context) {
		if service == nil {
			ctx.JSON(http.StatusServiceUnavailable, map[string]string{"error": "capability service unavailable"})
			return
		}
		var warnings []string
		if statusProvider != nil {
			if status := statusProvider(); status != nil && status.Outdated {
				warning := status.Message
				if strings.TrimSpace(warning) == "" {
					warning = "Gateway 契约版本需升级，请同步最新 contracts。"
				}
				ctx.SetHeader(contractStatusHeader, warning)
				warnings = append(warnings, warning)
			}
		}
		var req capabilityInvokeRequest
		if err := ctx.BindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid capability payload"})
			return
		}
		if req.Payload == nil {
			req.Payload = map[string]any{}
		}
		headers := collectCapabilityProxyHeaders(ctx)
		if module := proxyMockModule(headers); module != "" {
			warnings = append(warnings, "通过 X-PX-Use-Mock 请求 Mock 模块: "+module)
		}
		tenantUUID := strings.TrimSpace(ctx.Header("X-Tenant-UUID"))
		if tenantUUID != "" {
			if headers == nil {
				headers = make(map[string]string, 1)
			}
			headers["X-Tenant-UUID"] = tenantUUID
		}
		result, err := service.Invoke(ctx.Context(), capabilityinvoker.InvokeParams{
			CapabilityID: req.CapabilityID,
			Action:       req.Action,
			Payload:      req.Payload,
			Headers:      headers,
			RequestID:    ctx.Header("X-Request-ID"),
			TenantUUID:   tenantUUID,
		})
		if err != nil {
			writeCapabilityError(ctx, err, warnings)
			return
		}
		response := map[string]any{
			"traceId": result.TraceID,
			"status":  result.Status,
			"data":    result.Data,
		}
		if len(warnings) > 0 {
			response["warnings"] = warnings
		}
		if len(result.Raw) > 0 {
			response["raw"] = json.RawMessage(result.Raw)
		}
		if result.TraceID != "" {
			ctx.SetHeader("X-Trace-Id", result.TraceID)
		}
		ctx.JSON(http.StatusOK, response)
	}
}

func writeCapabilityError(ctx bootstrap.Context, err error, warnings []string) {
	invokeErr := &capabilityinvoker.InvokeError{}
	if !errors.As(err, &invokeErr) {
		ctx.JSON(http.StatusBadGateway, map[string]any{
			"error":   err.Error(),
			"traceId": "",
		})
		return
	}
	status := http.StatusBadGateway
	switch invokeErr.Category {
	case capabilityinvoker.ErrorCategoryValidation:
		status = http.StatusBadRequest
	case capabilityinvoker.ErrorCategoryUnauthorized:
		status = http.StatusUnauthorized
	case capabilityinvoker.ErrorCategoryRateLimited:
		status = http.StatusTooManyRequests
	default:
		if invokeErr.Status >= 400 {
			status = invokeErr.Status
		}
	}
	if invokeErr.TraceID != "" {
		ctx.SetHeader("X-Trace-Id", invokeErr.TraceID)
	}
	payload := map[string]any{
		"error": map[string]any{
			"code":    invokeErr.Code,
			"message": invokeErr.Message,
			"type":    invokeErr.Category,
		},
		"traceId": invokeErr.TraceID,
	}
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	ctx.JSON(status, payload)
}

func collectCapabilityProxyHeaders(ctx bootstrap.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	headers := make(map[string]string, len(capabilityProxyForwardHeaders))
	for _, name := range capabilityProxyForwardHeaders {
		value := strings.TrimSpace(ctx.Header(name))
		if value == "" {
			continue
		}
		headers[name] = value
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func proxyMockModule(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	return strings.TrimSpace(headers["X-PX-Use-Mock"])
}
