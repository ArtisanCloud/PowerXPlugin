package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
)

func TestRoutePathParam(t *testing.T) {
	root := newHTTPRouterRoot()
	r := &httpRouter{root: root}

	var captured string
	r.Handle(http.MethodGet, "/templates/:id", func(ctx bootstrap.Context) {
		captured = ctx.Param("id")
		RespondSuccess(ctx, http.StatusOK, map[string]string{"id": captured}, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/templates/42", nil)
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if captured != "42" {
		t.Fatalf("expected param 42, got %s", captured)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if data, ok := payload["data"].(map[string]any); !ok || data["id"] != "42" {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

func TestRespondError(t *testing.T) {
	root := newHTTPRouterRoot()
	r := &httpRouter{root: root}

	r.Handle(http.MethodGet, "/fail", func(ctx bootstrap.Context) {
		RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "bad input", map[string]string{"field": "id"})
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var payload struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Success {
		t.Fatalf("expected success=false")
	}
	if payload.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}
}

func TestGroupAndNotFound(t *testing.T) {
	t.Helper()
	root := newHTTPRouterRoot()
	r := &httpRouter{root: root}

	api := r.Group("/api")
	api.Handle(http.MethodGet, "/ping", func(ctx bootstrap.Context) {
		RespondSuccess(ctx, http.StatusOK, map[string]string{"status": "ok"}, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	notFound := httptest.NewRecorder()
	root.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing route, got %d", notFound.Code)
	}
}

func TestAttachAndFrameworkRoute(t *testing.T) {
	app := bootstrap.NewApp(&bootstrap.Config{Listen: ":0"})
	if err := AttachHTTPServer(app); err != nil {
		t.Fatalf("attach http server: %v", err)
	}
	RegisterFrameworkRoutes(app)

	httpRouter, ok := app.Router.(*httpRouter)
	if !ok {
		t.Fatalf("expected httpRouter, got %T", app.Router)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, HealthzPath, nil)
	httpRouter.root.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from healthz, got %d", rec.Code)
	}
}

func TestRegisterHelpersNil(t *testing.T) {
	RegisterFrameworkRoutes(nil)
	RegisterPluginRoutes(nil, nil)
}

func TestRegisterPluginRoutes(t *testing.T) {
	app := bootstrap.NewApp(&bootstrap.Config{Listen: ":0"})
	if err := AttachHTTPServer(app); err != nil {
		t.Fatalf("attach http server: %v", err)
	}

	RegisterPluginRoutes(app, func(rg bootstrap.Router) {
		rg.Handle(http.MethodGet, "/items", func(ctx bootstrap.Context) {
			RespondSuccess(ctx, http.StatusOK, map[string]string{"q": ctx.Query("q")}, "")
		})
	})

	httpRouter := app.Router.(*httpRouter)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items?q=test", nil)
	httpRouter.root.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if data, ok := payload["data"].(map[string]any); !ok || data["q"] != "test" {
		t.Fatalf("unexpected data payload: %v", payload)
	}
}

func TestJoinPathsAndSplit(t *testing.T) {
	tests := []struct {
		base string
		rel  string
		want string
	}{
		{"/api", "v1", "/api/v1"},
		{"/api/", "/v1/items/", "/api/v1/items/"},
		{"", "/healthz", "/healthz"},
		{"/", "/", "/"},
	}
	for _, tc := range tests {
		if got := joinPaths(tc.base, tc.rel); got != tc.want {
			t.Fatalf("joinPaths(%q,%q)=%q, want %q", tc.base, tc.rel, got, tc.want)
		}
	}

	segments := splitPath("/api/v1/items")
	if len(segments) != 3 || segments[0] != "api" || segments[2] != "items" {
		t.Fatalf("unexpected segments: %v", segments)
	}
}

func TestUseMiddlewareOrder(t *testing.T) {
	root := newHTTPRouterRoot()
	r := &httpRouter{root: root}
	var sequence []string

	r.Use()

	r.Use(func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			sequence = append(sequence, "mw1")
			if next != nil {
				next(ctx)
			}
			sequence = append(sequence, "mw2")
		}
	})

	r.Handle(http.MethodGet, "/hook", func(ctx bootstrap.Context) {
		sequence = append(sequence, "handler")
		RespondSuccess(ctx, http.StatusOK, nil, "")
	})

	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hook", nil))
	expected := []string{"mw1", "handler", "mw2"}
	for i := range expected {
		if sequence[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, sequence)
		}
	}
}

func TestMatchSegments(t *testing.T) {
	params, ok := matchSegments([]string{"templates", ":id", "revisions", ":rid"}, []string{"templates", "1", "revisions", "2"})
	if !ok {
		t.Fatalf("expected segments to match")
	}
	if params["id"] != "1" || params["rid"] != "2" {
		t.Fatalf("unexpected params: %v", params)
	}

	if _, ok := matchSegments([]string{"templates", ":id"}, []string{"notes", "1"}); ok {
		t.Fatalf("expected mismatch for different prefixes")
	}
}

func TestRespondHelpersWithRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Request-ID", "from-request")
	rec := httptest.NewRecorder()
	ctx := newHTTPContext(rec, req, nil)

	RespondSuccess(ctx, http.StatusOK, nil, "ok")
	var successPayload Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &successPayload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if successPayload.RequestID != "from-request" {
		t.Fatalf("expected request id from request header")
	}

	rec2 := httptest.NewRecorder()
	ctx2 := newHTTPContext(rec2, httptest.NewRequest(http.MethodGet, "/", nil), nil)
	ctx2.SetHeader("X-Request-ID", "generated")
	RespondError(ctx2, http.StatusBadRequest, "INVALID", "invalid", nil)
	var errPayload Envelope
	_ = json.Unmarshal(rec2.Body.Bytes(), &errPayload)
	if errPayload.RequestID != "generated" {
		t.Fatalf("expected request id from response header")
	}

	RespondSuccess(nil, http.StatusOK, nil, "")
	RespondError(nil, http.StatusInternalServerError, "", "", nil)
}

func TestAttachHTTPServerInvalid(t *testing.T) {
	if err := AttachHTTPServer(nil); err == nil {
		t.Fatalf("expected error when app is nil")
	}
}

func TestHTTPContextHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Test", "request")
	ctx := newHTTPContext(rec, req, nil)
	ctx.ctx = nil

	if ctx.Param("missing") != "" {
		t.Fatalf("expected empty param for nil map")
	}
	if ctx.Header("X-Test") != "request" {
		t.Fatalf("expected to read header from request")
	}
	ctx.SetContext(nil)
	if ctx.Context() != req.Context() {
		t.Fatalf("expected fallback to request context")
	}

	sanitize := sanitizePath("")
	if sanitize != "/" {
		t.Fatalf("sanitizePath empty expected '/' got %s", sanitize)
	}

	if out := splitPath("/"); len(out) != 0 {
		t.Fatalf("expected empty slice for root path")
	}
}

func TestBindJSON(t *testing.T) {
	root := newHTTPRouterRoot()
	r := &httpRouter{root: root}

	r.Handle(http.MethodPost, "/echo", func(ctx bootstrap.Context) {
		var payload map[string]string
		if err := ctx.BindJSON(&payload); err != nil {
			t.Fatalf("BindJSON failed: %v", err)
		}
		ctx.SetHeader("X-Test", "")
		RespondSuccess(ctx, http.StatusCreated, payload, "")
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"name":"demo"}`))
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if rec.Header().Get("X-Test") != "" {
		t.Fatalf("expected header removal")
	}
}

func TestHTTPContextHeadersAndContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	ctx := newHTTPContext(rec, req, map[string]string{"id": "7"})

	if ctx.Param("id") != "7" {
		t.Fatalf("expected param 7")
	}

	ctx.SetHeader("X-Test", "value")
	if ctx.Header("X-Test") != "value" {
		t.Fatalf("expected header to be set")
	}

	newCtx := context.WithValue(ctx.Context(), "key", "value")
	ctx.SetContext(newCtx)
	if ctx.Context().Value("key") != "value" {
		t.Fatalf("expected context to carry value")
	}
}
