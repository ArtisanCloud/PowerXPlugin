package templates

import (
	"context"
	"errors"
	"strings"

	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"
)

var (
	errTenantRequired = errors.New("tenant id required")
	errNotFound       = errors.New("template not found")
	errInvalidInput   = errors.New("invalid input")
)

// Service 提供模板业务编排逻辑。
type Service struct {
	repo *TemplateRepository
}

func NewService(repo *TemplateRepository) *Service {
	if repo == nil {
		repo = NewTemplateRepository()
	}
	return &Service{repo: repo}
}

// List 返回分页模板列表。
func (s *Service) List(ctx context.Context, query string, page, pageSize int) (*Page[[]*Template], error) {
	if _, ok := frameworkmw.TenantIDFromContext(ctx); !ok {
		return nil, errTenantRequired
	}
	res, err := s.repo.List(ctx, query, page, pageSize)
	if err != nil {
		return nil, translateError(err)
	}
	return res, nil
}

// Get 返回指定模板。
func (s *Service) Get(ctx context.Context, id uint64) (*Template, error) {
	if id == 0 {
		return nil, errInvalidInput
	}
	if _, ok := frameworkmw.TenantIDFromContext(ctx); !ok {
		return nil, errTenantRequired
	}
	tpl, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateError(err)
	}
	if tpl == nil {
		return nil, errNotFound
	}
	return tpl, nil
}

// Create 创建模板。
func (s *Service) Create(ctx context.Context, name, description, content string) (*Template, error) {
	if _, ok := frameworkmw.TenantIDFromContext(ctx); !ok {
		return nil, errTenantRequired
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(content) == "" {
		return nil, errInvalidInput
	}
	tpl := &Template{Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), Content: strings.TrimSpace(content)}
	result, err := s.repo.Create(ctx, tpl)
	if err != nil {
		return nil, translateError(err)
	}
	return result, nil
}

// Update 更新模板。
func (s *Service) Update(ctx context.Context, id uint64, name, description, content string) (*Template, error) {
	if id == 0 {
		return nil, errInvalidInput
	}
	if _, ok := frameworkmw.TenantIDFromContext(ctx); !ok {
		return nil, errTenantRequired
	}
	fields := map[string]string{
		"name":        strings.TrimSpace(name),
		"description": strings.TrimSpace(description),
		"content":     strings.TrimSpace(content),
	}
	result, err := s.repo.UpdateByID(ctx, id, fields)
	if err != nil {
		return nil, translateError(err)
	}
	return result, nil
}

// Delete 删除模板。
func (s *Service) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errInvalidInput
	}
	if _, ok := frameworkmw.TenantIDFromContext(ctx); !ok {
		return errTenantRequired
	}
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return translateError(err)
	}
	return nil
}

func translateError(err error) error {
	switch {
	case errors.Is(err, errTenantIDRequired):
		return errTenantRequired
	case errors.Is(err, errTemplateNotFound):
		return errNotFound
	default:
		return err
	}
}
