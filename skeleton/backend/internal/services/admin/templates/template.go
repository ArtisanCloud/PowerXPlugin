package templates

import (
	"context"
	"strings"

	dbm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository"
	trepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/template"
	"gorm.io/gorm"
)

type TemplateService struct {
	TemplateRepo *trepo.TemplateRepository
}

func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{TemplateRepo: trepo.NewTemplateRepository(db)}
}

func (s *TemplateService) List(
	ctx context.Context,
	q string,
	page, pageSize int,
) (*repository.Page[[]*dbm.Template], error) {
	cb := func(db *gorm.DB, opt interface{}) *gorm.DB {
		if kw, _ := opt.(string); strings.TrimSpace(kw) != "" {
			p := "%" + strings.TrimSpace(kw) + "%"
			db = db.Where("(name ILIKE ? OR description ILIKE ?)", p, p)
		}
		return db.Order("id DESC")
	}

	return s.TemplateRepo.FindPage(ctx, nil, page, pageSize, cb, q)
}

func (s *TemplateService) GetByID(ctx context.Context, id uint64) (*dbm.Template, error) {
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	tpl, err := s.TemplateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return tpl, nil
}

func (s *TemplateService) Create(
	ctx context.Context,
	name, description, content string,
) (*dbm.Template, error) {
	tpl := &dbm.Template{
		Name:        name,
		Description: description,
		Content:     content,
	}

	return s.TemplateRepo.Create(ctx, tpl)
}

func (s *TemplateService) Update(
	ctx context.Context,
	id uint64,
	name, description, content string,
) (*dbm.Template, error) {
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	fields := map[string]interface{}{
		"name":        name,
		"description": description,
		"content":     content,
	}
	return s.TemplateRepo.UpdateByID(ctx, id, fields)
}

func (s *TemplateService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return gorm.ErrInvalidData
	}
	return s.TemplateRepo.DeleteByID(ctx, id)
}
