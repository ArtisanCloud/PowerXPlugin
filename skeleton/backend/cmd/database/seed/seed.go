package seed

import (
	"context"
	"errors"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models"
	templatemodel "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/template"
	"gorm.io/gorm"
)

func SeedPluginData(ctx context.Context, db *gorm.DB) error {
	seedTemplates := []struct {
		Name        string
		Description string
		Content     string
	}{
		{
			Name:        "欢迎模板",
			Description: "展示如何在插件中定义第一条模板记录",
			Content:     "# 欢迎使用 PowerX Base 插件\n这是一个示例模板内容，您可以根据需要修改。",
		},
		{
			Name:        "周报模板",
			Description: "帮助团队快速整理一周的工作进展",
			Content:     "## 本周进展\n- 事项 A\n- 事项 B\n\n## 下周计划\n- 计划 A\n- 计划 B",
		},
	}

	const tenantID uint64 = 1

	ctxDB := db.WithContext(ctx)
	for _, tpl := range seedTemplates {
		var existing templatemodel.Template
		err := ctxDB.Where("tenant_id = ? AND name = ?", tenantID, tpl.Name).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			newTpl := templatemodel.Template{
				BaseModel:   models.BaseModel{TenantID: tenantID},
				Name:        tpl.Name,
				Description: tpl.Description,
				Content:     tpl.Content,
			}
			if err := ctxDB.Create(&newTpl).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			updates := map[string]interface{}{
				"description": tpl.Description,
				"content":     tpl.Content,
			}
			if err := ctxDB.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
