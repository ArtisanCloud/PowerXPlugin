package seed

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	agentregistry "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/agent_registry"
	templatemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	sampleskills "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/skills"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultTenantUUID = "00000000-0000-0000-0000-000000000001"
	pluginID          = "com.powerx.plugins.base"
)

func SeedPluginData(ctx context.Context, db *gorm.DB) error {
	if err := seedTemplateSkillPackage(ctx, db); err != nil {
		return err
	}
	seedTemplates := []struct {
		Name        string
		Description string
		Content     string
		Status      string
		Review      string
	}{
		{
			Name:        "欢迎模板",
			Description: "展示如何在插件中定义第一条模板记录",
			Content:     "# 欢迎使用 PowerX Base 插件\n这是一个示例模板内容，您可以根据需要修改。",
			Status:      "published",
			Review:      "approved",
		},
		{
			Name:        "周报模板",
			Description: "帮助团队快速整理一周的工作进展",
			Content:     "## 本周进展\n- 事项 A\n- 事项 B\n\n## 下周计划\n- 计划 A\n- 计划 B",
			Status:      "draft",
			Review:      "pending",
		},
	}

	now := time.Now()

	ctxDB := db.WithContext(ctx)
	for _, tpl := range seedTemplates {
		var existing templatemodel.Template
		err := ctxDB.Where("tenant_uuid = ? AND name = ?", defaultTenantUUID, tpl.Name).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			newTpl := templatemodel.Template{
				BaseModel:    models.BaseModel{TenantUuid: defaultTenantUUID},
				Name:         tpl.Name,
				Description:  tpl.Description,
				Content:      tpl.Content,
				Status:       tpl.Status,
				ReviewStatus: tpl.Review,
				ReviewedBy:   "seed",
			}
			if tpl.Review == "approved" {
				newTpl.ReviewedAt = &now
			}
			if tpl.Status == "published" {
				newTpl.PublishChannel = "mini-app"
				newTpl.PublishedAt = &now
			}
			if err := ctxDB.Create(&newTpl).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			updates := map[string]interface{}{
				"description":   tpl.Description,
				"content":       tpl.Content,
				"status":        tpl.Status,
				"review_status": tpl.Review,
			}
			if tpl.Review == "approved" {
				updates["reviewed_by"] = "seed"
				updates["reviewed_at"] = &now
			} else {
				updates["reviewed_by"] = ""
				updates["reviewed_at"] = nil
			}
			if tpl.Status == "published" {
				updates["publish_channel"] = "mini-app"
				updates["published_at"] = &now
			} else {
				updates["publish_channel"] = ""
				updates["published_at"] = nil
			}
			if err := ctxDB.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func seedTemplateSkillPackage(ctx context.Context, db *gorm.DB) error {
	pkg, err := sampleskills.LoadTemplatePackage()
	if err != nil {
		return err
	}
	intentExamples := mustJSON(pkg.Manifest.IntentExamples)
	inputSchema := mustJSON(pkg.InputSchema)
	outputSchema := mustJSON(pkg.OutputSchema)
	frontmatter := mustJSON(pkg.Frontmatter)
	promptSpec := mustJSON(map[string]any{
		"source":            "skill_package",
		"body_markdown":     pkg.BodyMarkdown,
		"response_guidance": pkg.Manifest.ResponseGuidance,
	})
	executor := mustJSON(pkg.Manifest.Executor)
	ctxDB := db.WithContext(ctx)
	var skill agentregistry.PluginSkill
	err = ctxDB.Where("tenant_uuid = ? AND plugin_id = ? AND plugin_skill_id = ? AND version = ?", defaultTenantUUID, pluginID, pkg.Manifest.SkillID, pkg.Manifest.Version).First(&skill).Error
	updates := map[string]any{
		"title":              pkg.Manifest.Title,
		"description":        pkg.Manifest.Description,
		"intent_examples":    datatypes.JSON(intentExamples),
		"input_schema":       datatypes.JSON(inputSchema),
		"output_schema":      datatypes.JSON(outputSchema),
		"prompt_spec":        datatypes.JSON(promptSpec),
		"executor":           datatypes.JSON(executor),
		"capability":         pkg.Manifest.Executor.Capability,
		"source_format":      pkg.SourceFormat,
		"package_path":       pkg.PackagePath,
		"skill_md_path":      pkg.SkillMDPath,
		"raw_markdown":       pkg.RawMarkdown,
		"frontmatter_json":   datatypes.JSON(frontmatter),
		"body_markdown":      pkg.BodyMarkdown,
		"package_checksum":   pkg.Checksum,
		"checksum":           pkg.Checksum,
		"sync_error_code":    "",
		"sync_error_message": "",
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		skill = agentregistry.PluginSkill{
			BaseModel:       models.BaseModel{TenantUuid: defaultTenantUUID},
			PluginSkillID:   pkg.Manifest.SkillID,
			PluginID:        pluginID,
			Version:         pkg.Manifest.Version,
			Title:           pkg.Manifest.Title,
			Description:     pkg.Manifest.Description,
			IntentExamples:  datatypes.JSON(intentExamples),
			InputSchema:     datatypes.JSON(inputSchema),
			OutputSchema:    datatypes.JSON(outputSchema),
			PromptSpec:      datatypes.JSON(promptSpec),
			Executor:        datatypes.JSON(executor),
			Capability:      pkg.Manifest.Executor.Capability,
			SourceFormat:    pkg.SourceFormat,
			PackagePath:     pkg.PackagePath,
			SkillMDPath:     pkg.SkillMDPath,
			RawMarkdown:     pkg.RawMarkdown,
			FrontmatterJSON: datatypes.JSON(frontmatter),
			BodyMarkdown:    pkg.BodyMarkdown,
			PackageChecksum: pkg.Checksum,
			Checksum:        pkg.Checksum,
			SyncStatus:      agentregistry.SyncStatusDraft,
		}
		if err := ctxDB.Create(&skill).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if skill.SyncStatus == "" {
			updates["sync_status"] = agentregistry.SyncStatusDraft
		}
		if err := ctxDB.Model(&skill).Updates(updates).Error; err != nil {
			return err
		}
	}

	pluginSkillIDs := []string{pkg.Manifest.SkillID}
	var agent agentregistry.PluginAgent
	err = ctxDB.Where("tenant_uuid = ? AND plugin_id = ? AND plugin_agent_id = ?", defaultTenantUUID, pluginID, "template-agent").First(&agent).Error
	agentUpdates := map[string]any{
		"agent_key":          "powerxplugin.template.agent",
		"name":               "模板智能体",
		"description":        "面向插件开发者和管理员的 PowerXPlugin 模板对象管理智能体，负责解释并执行模板对象的创建、查询、更新、删除和列表等任务。",
		"persona":            "你是 PowerXPlugin 模板对象管理助手，服务对象是插件开发者和插件管理员。你负责围绕模板对象进行自然语言对话、能力解释、参数澄清和任务执行。回答时应先理解用户当前问题，再基于当前绑定 Skill 的真实 metadata 说明能力或发起执行；不要编造未绑定能力，不要暴露内部 skill_id、executor path、schema 字段名。",
		"prompt_seed":        "当用户询问你是谁或能做什么时，请以模板对象管理助手身份回答，并只基于当前已绑定 Skill 的真实 metadata 介绍能力。能力介绍应先说明服务对象，再概括模板创建、查询、更新、删除和列表；推荐先测试创建模板。用户要求执行时，按 Skill response_guidance 和 input_schema 判断缺参并追问，参数足够后调用绑定 Skill。",
		"plugin_skill_ids":   datatypes.JSON(mustJSON(pluginSkillIDs)),
		"sync_error_code":    "",
		"sync_error_message": "",
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		agent = agentregistry.PluginAgent{
			BaseModel:      models.BaseModel{TenantUuid: defaultTenantUUID},
			PluginAgentID:  "template-agent",
			PluginID:       pluginID,
			AgentKey:       "powerxplugin.template.agent",
			Name:           "模板智能体",
			Description:    "面向插件开发者和管理员的 PowerXPlugin 模板对象管理智能体，负责解释并执行模板对象的创建、查询、更新、删除和列表等任务。",
			Persona:        "你是 PowerXPlugin 模板对象管理助手，服务对象是插件开发者和插件管理员。你负责围绕模板对象进行自然语言对话、能力解释、参数澄清和任务执行。回答时应先理解用户当前问题，再基于当前绑定 Skill 的真实 metadata 说明能力或发起执行；不要编造未绑定能力，不要暴露内部 skill_id、executor path、schema 字段名。",
			PromptSeed:     "当用户询问你是谁或能做什么时，请以模板对象管理助手身份回答，并只基于当前已绑定 Skill 的真实 metadata 介绍能力。能力介绍应先说明服务对象，再概括模板创建、查询、更新、删除和列表；推荐先测试创建模板。用户要求执行时，按 Skill response_guidance 和 input_schema 判断缺参并追问，参数足够后调用绑定 Skill。",
			PluginSkillIDs: datatypes.JSON(mustJSON(pluginSkillIDs)),
			PowerXSkillIDs: datatypes.JSON(mustJSON([]string{})),
			SyncStatus:     agentregistry.SyncStatusDraft,
		}
		if err := ctxDB.Create(&agent).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if agent.SyncStatus == "" {
			agentUpdates["sync_status"] = agentregistry.SyncStatusDraft
		}
		return ctxDB.Model(&agent).Updates(agentUpdates).Error
	}

	return nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return []byte(`{}`)
	}
	return b
}
