package templates

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"
	"github.com/powerx-plugin/framework/backend/go/router"
)

type handlerStubContext struct {
	params      map[string]string
	query       map[string]string
	headers     map[string]string
	status      int
	payload     any
	ctx         context.Context
	method      string
	bindPayload any
	bindErr     error
}

func newHandlerStub() *handlerStubContext {
	return &handlerStubContext{
		params:  make(map[string]string),
		query:   make(map[string]string),
		headers: make(map[string]string),
		ctx:     context.Background(),
		method:  http.MethodGet,
	}
}

func (s *handlerStubContext) Param(name string) string {
	return s.params[name]
}

func (s *handlerStubContext) Query(name string) string {
	return s.query[name]
}

func (s *handlerStubContext) BindJSON(v any) error {
	if s.bindErr != nil {
		return s.bindErr
	}
	if s.bindPayload == nil {
		return nil
	}
	data, err := json.Marshal(s.bindPayload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (s *handlerStubContext) JSON(status int, v any) {
	s.status = status
	s.payload = v
}

func (s *handlerStubContext) Status(code int) {
	s.status = code
}

func (s *handlerStubContext) Header(name string) string {
	if v, ok := s.headers[name]; ok {
		return v
	}
	return ""
}

func (s *handlerStubContext) SetHeader(name, value string) {
	if value == "" {
		delete(s.headers, name)
		return
	}
	s.headers[name] = value
}

func (s *handlerStubContext) Method() string {
	if s.method == "" {
		return http.MethodGet
	}
	return s.method
}

func (s *handlerStubContext) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *handlerStubContext) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func ctxWithTenantID(id uint64) context.Context {
	return frameworkmw.WithTenantID(context.Background(), id)
}

func extractEnvelope(t *testing.T, payload any) router.Envelope {
	t.Helper()
	env, ok := payload.(router.Envelope)
	if !ok {
		t.Fatalf("expected router.Envelope, got %#v", payload)
	}
	return env
}

func TestHandlerListSuccess(t *testing.T) {
	repo := NewTemplateRepository()
	svc := NewService(repo)
	ctx := ctxWithTenantID(1)
	if _, err := svc.Create(ctx, "Demo", "Desc", "Content"); err != nil {
		t.Fatalf("seed create: %v", err)
	}

	h := NewHandler(svc)
	stub := newHandlerStub()
	stub.ctx = ctx
	stub.query["page"] = "1"
	stub.query["page_size"] = "5"

	h.List()(stub)

	if stub.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", stub.status)
	}
	env := extractEnvelope(t, stub.payload)
	if !env.Success {
		t.Fatalf("expected success envelope: %#v", env)
	}
	page, ok := env.Data.(*Page[[]*Template])
	if !ok {
		t.Fatalf("expected Page data, got %#v", env.Data)
	}
	if len(page.List) != 1 {
		t.Fatalf("expected 1 template, got %d", len(page.List))
	}
}

func TestHandlerListUnauthorized(t *testing.T) {
	h := NewHandler(NewService(NewTemplateRepository()))
	stub := newHandlerStub()

	h.List()(stub)

	if stub.status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", stub.status)
	}
	env := extractEnvelope(t, stub.payload)
	if env.Success || env.Error == nil || env.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
}

func TestHandlerCreate(t *testing.T) {
	h := NewHandler(NewService(NewTemplateRepository()))

	// invalid JSON body
	stub := newHandlerStub()
	stub.bindErr = errors.New("decode error")
	h.Create()(stub)
	if stub.status != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", stub.status)
	}

	// missing tenant
	stubMissingTenant := newHandlerStub()
	stubMissingTenant.method = http.MethodPost
	stubMissingTenant.bindPayload = map[string]string{"name": "Demo", "content": "Body"}
	h.Create()(stubMissingTenant)
	if stubMissingTenant.status != http.StatusUnauthorized {
		t.Fatalf("expected 401 without tenant, got %d", stubMissingTenant.status)
	}

	// success
	successStub := newHandlerStub()
	successStub.method = http.MethodPost
	successStub.bindPayload = map[string]string{"name": " Demo ", "description": " Desc ", "content": " Content "}
	successStub.ctx = ctxWithTenantID(2)
	h.Create()(successStub)

	if successStub.status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", successStub.status)
	}
	env := extractEnvelope(t, successStub.payload)
	if !env.Success {
		t.Fatalf("expected success envelope: %#v", env)
	}
	tpl, ok := env.Data.(*Template)
	if !ok {
		t.Fatalf("expected template data, got %#v", env.Data)
	}
	if tpl.Name != "Demo" || tpl.Content != "Content" {
		t.Fatalf("unexpected template fields: %+v", tpl)
	}
}

func TestHandlerGetAndUpdateErrors(t *testing.T) {
	repo := NewTemplateRepository()
	svc := NewService(repo)
	ctx := ctxWithTenantID(5)
	created, err := svc.Create(ctx, "Demo", "", "Body")
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	h := NewHandler(svc)

	// invalid id
	invalidID := newHandlerStub()
	invalidID.params["id"] = "abc"
	h.Get()(invalidID)
	if invalidID.status != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", invalidID.status)
	}

	// not found
	notFound := newHandlerStub()
	notFound.ctx = ctx
	notFound.params["id"] = "999"
	h.Get()(notFound)
	if notFound.status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", notFound.status)
	}

	// update success
	updateStub := newHandlerStub()
	updateStub.ctx = ctx
	updateStub.method = http.MethodPut
	updateStub.params["id"] = strconv.FormatUint(created.ID, 10)
	updateStub.bindPayload = map[string]string{"name": "Updated", "content": "New"}
	h.Update()(updateStub)
	if updateStub.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", updateStub.status)
	}

	// update invalid body
	updateInvalid := newHandlerStub()
	updateInvalid.ctx = ctx
	updateInvalid.method = http.MethodPut
	updateInvalid.params["id"] = strconv.FormatUint(created.ID, 10)
	updateInvalid.bindErr = errors.New("decode failure")
	h.Update()(updateInvalid)
	if updateInvalid.status != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", updateInvalid.status)
	}
}

func TestHandlerDelete(t *testing.T) {
	repo := NewTemplateRepository()
	svc := NewService(repo)
	ctx := ctxWithTenantID(42)
	created, err := svc.Create(ctx, "Demo", "", "Body")
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	h := NewHandler(svc)

	// invalid id
	invalid := newHandlerStub()
	invalid.params["id"] = "abc"
	h.Delete()(invalid)
	if invalid.status != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", invalid.status)
	}

	// success delete
	okStub := newHandlerStub()
	okStub.ctx = ctx
	okStub.params["id"] = strconv.FormatUint(created.ID, 10)
	h.Delete()(okStub)
	if okStub.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", okStub.status)
	}
	env := extractEnvelope(t, okStub.payload)
	okMap, ok := env.Data.(map[string]bool)
	if !ok || !okMap["ok"] {
		t.Fatalf("expected ok response, got %#v", env.Data)
	}

	// deleting again -> not found
	notFound := newHandlerStub()
	notFound.ctx = ctx
	notFound.params["id"] = strconv.FormatUint(created.ID, 10)
	h.Delete()(notFound)
	if notFound.status != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", notFound.status)
	}
}
