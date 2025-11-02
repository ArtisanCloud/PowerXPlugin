package templates

import (
	"context"
	"errors"
	"testing"

	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"
)

func ctxWithTenant(t uint64) context.Context {
	return frameworkmw.WithTenantID(context.Background(), t)
}

func TestServiceLifecycle(t *testing.T) {
	repo := NewTemplateRepository()
	svc := NewService(repo)

	ctx := ctxWithTenant(100)
	created, err := svc.Create(ctx, "Demo", "Desc", "Content")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 || created.TenantID != 100 {
		t.Fatalf("unexpected created entity: %+v", created)
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.Name != "Demo" {
		t.Fatalf("unexpected fetched: %+v", fetched)
	}

	updated, err := svc.Update(ctx, created.ID, "Demo2", "Desc2", "Content2")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Demo2" {
		t.Fatalf("expected name to update: %+v", updated)
	}

	page, err := svc.List(ctx, "demo", 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.List) != 1 || page.Total != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err = svc.Get(ctx, created.ID); !errors.Is(err, errNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestServiceEdgeCases(t *testing.T) {
	svc := NewService(NewTemplateRepository())
	if _, err := svc.Create(context.Background(), "", "", ""); !errors.Is(err, errTenantRequired) {
		t.Fatalf("expected tenant required error")
	}

	ctx := ctxWithTenant(1)
	if _, err := svc.Create(ctx, "", "", ""); !errors.Is(err, errInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}

	svc = NewService(NewTemplateRepository())
	ctx = ctxWithTenant(1)
	if _, err := svc.Create(ctx, "demo", "", "content"); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if _, err := svc.List(context.Background(), "", 1, 10); !errors.Is(err, errTenantRequired) {
		t.Fatalf("expected tenant required on list")
	}

	if err := svc.Delete(ctx, 999); !errors.Is(err, errNotFound) {
		t.Fatalf("expected delete not found")
	}

	if _, err := svc.Update(context.Background(), 1, "", "", ""); !errors.Is(err, errTenantRequired) {
		t.Fatalf("expected tenant required on update")
	}
}
