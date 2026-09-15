package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	entmodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/agent_registry"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const testTenantUUID = "6b5d0240-9920-46da-b707-88200e0f51ea"
const testOriginTenantUUID = "00000000-0000-0000-0000-000000000001"

type stubGateway struct{}

func newAgentRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	entmodels.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbm.PluginSkill{}, &dbm.PluginAgent{}))
	return db
}

func (stubGateway) Enabled() bool { return true }
func (stubGateway) Invoke(context.Context, gateway.InvokeParams) (*gateway.InvokeResult, error) {
	return nil, nil
}
func (stubGateway) ListPlatformCapabilityCatalog(context.Context, gateway.ListPlatformCapabilityCatalogOptions) ([]gateway.PlatformCapabilityCatalogRecord, error) {
	return nil, nil
}
func (stubGateway) ListKnowledgeSpaces(context.Context, gateway.KnowledgeSpaceListOptions) ([]gateway.KnowledgeSpaceRuntimeRecord, error) {
	return nil, nil
}
func (stubGateway) CreateKnowledgeSpace(context.Context, gateway.KnowledgeSpaceCreateParams) (*gateway.KnowledgeSpaceRecord, error) {
	return nil, nil
}
func (stubGateway) RetireKnowledgeSpace(context.Context, gateway.KnowledgeSpaceRetireParams) (*gateway.KnowledgeSpaceRecord, error) {
	return nil, nil
}
func (stubGateway) DeleteKnowledgeSpace(context.Context, gateway.KnowledgeSpaceDeleteParams) error {
	return nil
}
func (stubGateway) ResolveGatewayTenantUUID(context.Context) (string, error) {
	return testTenantUUID, nil
}
func (stubGateway) ListAgents(context.Context, string) ([]gateway.AgentRecord, error) {
	return []gateway.AgentRecord{
		{ID: float64(1), UUID: "00000000-0000-4000-8000-000000000001", Key: "system.default", Name: "System Default Agent", Status: "active"},
		{ID: float64(2), UUID: "00000000-0000-4000-8000-000000000002", Key: "template.crud", Name: "Template CRUD Agent", Status: "active"},
	}, nil
}
func (stubGateway) GetAgent(context.Context, string) (*gateway.AgentRecord, error) { return nil, nil }
func (stubGateway) SyncPluginSkill(context.Context, gateway.PluginSkillSyncParams) (*gateway.PluginSkillSyncResult, error) {
	return nil, nil
}
func (stubGateway) SyncPluginAgent(context.Context, gateway.PluginAgentSyncParams) (*gateway.PluginAgentSyncResult, error) {
	return nil, nil
}
func (stubGateway) RegisterCatalog(context.Context, *capabilities.CatalogSnapshot, []capabilities.ProtocolAsset) error {
	return nil
}

func (stubGateway) Close() error { return nil }

func TestListAgentsUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/agents?env=dev", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "System Default Agent")
	require.Contains(t, w.Body.String(), "template.crud")
}

func TestGetAgentEffectivePermissionsUsesSkillCapabilityRBAC(t *testing.T) {
	t.Setenv("POWERX_PLUGIN_REGISTRATION_MODE", "installed")
	gin.SetMode(gin.TestMode)
	db := newAgentRouteTestDB(t)
	executor := map[string]any{
		"type":       "capability",
		"capability": "powerxplugin.template",
		"action_map": map[string]string{
			"create": "com.powerx.plugins.base.template.create",
			"delete": "com.powerx.plugins.base.template.delete",
		},
	}
	require.NoError(t, db.Create(&dbm.PluginSkill{
		BaseModel:     entmodels.BaseModel{TenantUuid: testTenantUUID},
		PluginSkillID: "powerxplugin.template.basic",
		PluginID:      "com.powerx.plugins.base",
		PowerXSkillID: "powerxplugin.template.basic",
		Version:       "1.0.0",
		Title:         "Template",
		Executor:      datatypes.JSON(mustMarshalTestJSON(t, executor)),
		Capability:    "powerxplugin.template",
		SyncStatus:    dbm.SyncStatusSynced,
	}).Error)
	require.NoError(t, db.Create(&dbm.PluginAgent{
		BaseModel:       entmodels.BaseModel{TenantUuid: testTenantUUID},
		PluginAgentID:   "template-agent",
		PluginID:        "com.powerx.plugins.base",
		PowerXAgentUUID: "00000000-0000-4000-8000-000000000002",
		AgentKey:        "powerxplugin.template.agent",
		Name:            "Template Agent",
		PluginSkillIDs:  datatypes.JSON(mustMarshalTestJSON(t, []string{"powerxplugin.template.basic"})),
		PowerXSkillIDs:  datatypes.JSON(mustMarshalTestJSON(t, []string{"powerxplugin.template.basic"})),
		SyncStatus:      dbm.SyncStatusSynced,
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authx.SetTenantContext(c, authx.TenantContext{
			TenantUUID:  testTenantUUID,
			UserID:      42,
			Permissions: []string{"template:create"},
		})
		c.Next()
	})
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{DB: db, CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/agents/00000000-0000-4000-8000-000000000002/effective-permissions", nil)
	req.Header.Set("tenant_uuid", testTenantUUID)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"can_use_agent":true`)
	require.Contains(t, w.Body.String(), `"action":"create"`)
	require.Contains(t, w.Body.String(), `"allowed":true`)
	require.Contains(t, w.Body.String(), `"action":"delete"`)
	require.Contains(t, w.Body.String(), `"deny_code":"permission_denied"`)
	require.Contains(t, w.Body.String(), `"template:create"`)
}

func TestGetAgentEffectivePermissionsLooksUpAgentByGatewayTenantAndAuthorizesUserTenant(t *testing.T) {
	t.Setenv("POWERX_PLUGIN_REGISTRATION_MODE", "installed")
	gin.SetMode(gin.TestMode)
	db := newAgentRouteTestDB(t)
	executor := map[string]any{
		"type":       "capability",
		"capability": "powerxplugin.template",
		"action_map": map[string]string{
			"create": "com.powerx.plugins.base.template.create",
		},
	}
	require.NoError(t, db.Create(&dbm.PluginSkill{
		BaseModel:     entmodels.BaseModel{TenantUuid: testTenantUUID},
		PluginSkillID: "powerxplugin.template.basic",
		PluginID:      "com.powerx.plugins.base",
		PowerXSkillID: "powerxplugin.template.basic",
		Version:       "1.0.0",
		Title:         "Template",
		Executor:      datatypes.JSON(mustMarshalTestJSON(t, executor)),
		Capability:    "powerxplugin.template",
		SyncStatus:    dbm.SyncStatusSynced,
	}).Error)
	require.NoError(t, db.Create(&dbm.PluginAgent{
		BaseModel:       entmodels.BaseModel{TenantUuid: testTenantUUID},
		PluginAgentID:   "template-agent",
		PluginID:        "com.powerx.plugins.base",
		PowerXAgentUUID: "00000000-0000-4000-8000-000000000002",
		AgentKey:        "powerxplugin.template.agent",
		Name:            "Template Agent",
		PluginSkillIDs:  datatypes.JSON(mustMarshalTestJSON(t, []string{"powerxplugin.template.basic"})),
		PowerXSkillIDs:  datatypes.JSON(mustMarshalTestJSON(t, []string{"powerxplugin.template.basic"})),
		SyncStatus:      dbm.SyncStatusSynced,
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authx.SetTenantContext(c, authx.TenantContext{
			TenantUUID:  testOriginTenantUUID,
			UserID:      42,
			Permissions: []string{"template:create"},
		})
		c.Next()
	})
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{DB: db, CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/agents/00000000-0000-4000-8000-000000000002/effective-permissions", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"can_use_agent":true`)
	require.Contains(t, w.Body.String(), `"tenant_uuid":"`+testOriginTenantUUID+`"`)
	require.Contains(t, w.Body.String(), `"allowed":true`)
}

func mustMarshalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}
