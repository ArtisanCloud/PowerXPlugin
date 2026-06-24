package skills

import (
	"context"
	"testing"

	runtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	entmodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAgentBridgeExecutorReceivesCompleteContext(t *testing.T) {
	inv := runtime.PluginSkillInvocation{
		SkillID: TemplateSkillID,
		Version: "1.0.0",
		Input: map[string]any{"action": "create", "template": map[string]any{
			"title":       "测试模板",
			"description": "用于验证插件 CRUD 的基础模板对象",
			"content":     "这是一条测试内容",
		}},
		Context: runtime.PluginSkillInvocationContext{
			TenantUUID: "tenant_001",
			UserUUID:   "user_001",
			AgentID:    "agent_001",
			SessionID:  "session_001",
			MessageID:  "message_001",
			SkillID:    TemplateSkillID,
			TraceID:    "trace_001",
			Channel:    "web",
			Locale:     "zh-CN",
			Capability: "powerxplugin.template",
		},
	}

	entmodels.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbtemplate.Template{}))
	reg, err := NewTemplateRegistry(db)
	require.NoError(t, err)

	result, err := reg.Invoke(context.Background(), inv)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "trace_001", result.TraceID)
	require.Equal(t, "模板已创建", result.Message)
}
