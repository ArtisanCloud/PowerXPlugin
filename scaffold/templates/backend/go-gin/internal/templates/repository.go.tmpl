package templates

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"
)

var (
	errTenantIDRequired = errors.New("tenant id is required")
	errTemplateNotFound = errors.New("template not found")
)

// Page represents a paginated result.
type Page[T any] struct {
	List      T     `json:"list"`
	PageIndex int   `json:"page_index"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
}

// baseRepository simulates the constitution BaseRepository behaviour for the scaffold.
type baseRepository[T any] struct{}

func (b *baseRepository[T]) WithTenantTx(ctx context.Context, tenantID uint64, fn func(context.Context) error) error {
	if fn == nil {
		return errors.New("transaction callback is required")
	}
	ctx = frameworkmw.WithTenantID(ctx, tenantID)
	return fn(ctx)
}

// TemplateRepository stores templates in-memory by tenant.
type TemplateRepository struct {
	*baseRepository[Template]

	mu      sync.RWMutex
	store   map[uint64]map[uint64]*Template
	seq     map[uint64]uint64
	nowFunc func() time.Time
}

func NewTemplateRepository() *TemplateRepository {
	return &TemplateRepository{
		baseRepository: &baseRepository[Template]{},
		store:          make(map[uint64]map[uint64]*Template),
		seq:            make(map[uint64]uint64),
		nowFunc:        time.Now,
	}
}

func (r *TemplateRepository) List(ctx context.Context, query string, page, pageSize int) (*Page[[]*Template], error) {
	tenantID, ok := frameworkmw.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errTenantIDRequired
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*Template, 0)
	if tenantEntries, ok := r.store[tenantID]; ok {
		for _, tpl := range tenantEntries {
			items = append(items, cloneTemplate(tpl))
		}
	}

	if strings.TrimSpace(query) != "" {
		needle := strings.ToLower(strings.TrimSpace(query))
		filtered := make([]*Template, 0, len(items))
		for _, tpl := range items {
			if strings.Contains(strings.ToLower(tpl.Name), needle) || strings.Contains(strings.ToLower(tpl.Description), needle) {
				filtered = append(filtered, tpl)
			}
		}
		items = filtered
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })

	total := len(items)
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	result := make([]*Template, end-start)
	copy(result, items[start:end])

	return &Page[[]*Template]{
		List:      result,
		PageIndex: page,
		PageSize:  pageSize,
		Total:     int64(total),
	}, nil
}

func (r *TemplateRepository) FindByID(ctx context.Context, id uint64) (*Template, error) {
	tenantID, ok := frameworkmw.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errTenantIDRequired
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if tenantEntries, ok := r.store[tenantID]; ok {
		if tpl, ok := tenantEntries[id]; ok {
			return cloneTemplate(tpl), nil
		}
	}
	return nil, nil
}

func (r *TemplateRepository) Create(ctx context.Context, tpl *Template) (*Template, error) {
	tenantID, ok := frameworkmw.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errTenantIDRequired
	}

	var created *Template
	err := r.baseRepository.WithTenantTx(ctx, tenantID, func(txCtx context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		if tpl.TenantID != 0 && tpl.TenantID != tenantID {
			return errors.New("tenant mismatch")
		}

		r.seq[tenantID]++
		tpl.ID = r.seq[tenantID]
		tpl.TenantID = tenantID
		now := r.nowFunc().UTC()
		tpl.CreatedAt = now
		tpl.UpdatedAt = now

		if _, ok := r.store[tenantID]; !ok {
			r.store[tenantID] = make(map[uint64]*Template)
		}
		copy := cloneTemplate(tpl)
		r.store[tenantID][tpl.ID] = copy
		created = cloneTemplate(copy)
		return nil
	})

	return created, err
}

func (r *TemplateRepository) UpdateByID(ctx context.Context, id uint64, fields map[string]string) (*Template, error) {
	tenantID, ok := frameworkmw.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errTenantIDRequired
	}

	var updated *Template
	err := r.baseRepository.WithTenantTx(ctx, tenantID, func(txCtx context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		tenantEntries, ok := r.store[tenantID]
		if !ok {
			return errTemplateNotFound
		}
		existing, ok := tenantEntries[id]
		if !ok {
			return errTemplateNotFound
		}

		if v, ok := fields["name"]; ok {
			existing.Name = v
		}
		if v, ok := fields["description"]; ok {
			existing.Description = v
		}
		if v, ok := fields["content"]; ok {
			existing.Content = v
		}
		existing.UpdatedAt = r.nowFunc().UTC()
		updated = cloneTemplate(existing)
		return nil
	})

	return updated, err
}

func (r *TemplateRepository) DeleteByID(ctx context.Context, id uint64) error {
	tenantID, ok := frameworkmw.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return errTenantIDRequired
	}

	return r.baseRepository.WithTenantTx(ctx, tenantID, func(txCtx context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		tenantEntries, ok := r.store[tenantID]
		if !ok {
			return errTemplateNotFound
		}
		if _, ok := tenantEntries[id]; !ok {
			return errTemplateNotFound
		}
		delete(tenantEntries, id)
		return nil
	})
}

func cloneTemplate(src *Template) *Template {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}
