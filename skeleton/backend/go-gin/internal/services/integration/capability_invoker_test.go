package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	domain "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/templates"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCapabilityInvokerComposeCreatesTemplate(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	broker := stream.NewBroker()
	invoker := NewCapabilityInvoker(templateSvc, broker, logrus.New().WithField("test", "compose"), nil)

	sessionID := "sess-compose"
	eventsCh, cleanup := broker.Subscribe(sessionID)
	defer cleanup()

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-compose",
		ToolScope:  "agent.template.compose",
		PayloadRef: `{"draft":{"name":"Doc","description":"desc","content":"body"},"publish_channel":"global","review":{"reviewer":"qa-bot","comment":"looks good"},"cleanup":{"reason":"archived after publish"}}`,
		Metadata: map[string]any{
			"capability_id": (&templateComposeHandler{}).CapabilityID(),
			"session_id":    sessionID,
		},
	}

	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.NotNil(t, result)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &payload))
	require.Equal(t, srvtemplates.TemplateStatusArchived, payload["status"])
	require.Equal(t, srvtemplates.TemplateReviewApproved, payload["review_status"])
	lifecycle, ok := payload["lifecycle"].([]interface{})
	require.True(t, ok)
	require.Len(t, lifecycle, 4)

	ctx := authx.ContextWithTenantUUID(context.Background(), envelope.TenantUuid)
	page, err := templateSvc.List(ctx, "", 1, 10)
	require.NoError(t, err)
	require.Len(t, page.List, 1)
	require.Equal(t, "Doc", page.List[0].Name)
	require.Equal(t, srvtemplates.TemplateStatusArchived, page.List[0].Status)
	require.Equal(t, srvtemplates.TemplateReviewApproved, page.List[0].ReviewStatus)
	require.NotNil(t, page.List[0].CleanedAt)

	received := collectEvents(t, eventsCh, 5)
	require.Len(t, received, 5)
	require.Equal(t, "draft.created", received[0].Type)
	require.Equal(t, "template.review.completed", received[1].Type)
	require.Equal(t, "publish.status", received[2].Type)
	require.Equal(t, "template.publish.completed", received[3].Type)
	require.Equal(t, "template.cleanup.completed", received[4].Type)
}

func TestCapabilityInvokerAuditUpdatesTemplate(t *testing.T) {
	db, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-audit")
	require.NoError(t, db.WithContext(ctx).AutoMigrate(&dbtemplate.Template{}))
	tpl, err := templateSvc.Create(ctx, "AuditDoc", "old", "init")
	require.NoError(t, err)

	broker := stream.NewBroker()
	invoker := NewCapabilityInvoker(templateSvc, broker, logrus.New().WithField("test", "audit"), nil)
	sessionID := "sess-audit"
	eventsCh, cleanup := broker.Subscribe(sessionID)
	defer cleanup()

	payload := `{"filters":{"page":1,"page_size":5},"update_payload":{"description":"new desc","content":"updated body"}}`
	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-audit",
		ToolScope:  "agent.template.audit",
		PayloadRef: payload,
		Metadata: map[string]any{
			"capability_id": (&templateAuditHandler{}).CapabilityID(),
			"session_id":    sessionID,
		},
	}

	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &resp))
	require.Equal(t, true, resp["updated"])
	require.Equal(t, fmt.Sprintf("%d", tpl.ID), resp["selected_template_id"])

	updated, err := templateSvc.GetByID(ctx, tpl.ID)
	require.NoError(t, err)
	require.Equal(t, "new desc", updated.Description)
	require.Equal(t, "updated body", updated.Content)

	received := collectEvents(t, eventsCh, 1)
	require.NotEmpty(t, received)
	require.Equal(t, "audit.template.updated", received[0].Type)
}

func TestCapabilityInvokerQualityDistributeClonesTemplate(t *testing.T) {
	db, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-quality")
	require.NoError(t, db.WithContext(ctx).AutoMigrate(&dbtemplate.Template{}))
	_, err := templateSvc.Create(ctx, "BaseDoc", "origin", "initial content for qa")
	require.NoError(t, err)

	broker := stream.NewBroker()
	invoker := NewCapabilityInvoker(templateSvc, broker, logrus.New().WithField("test", "quality"), nil)
	sessionID := "sess-quality"
	eventsCh, cleanup := broker.Subscribe(sessionID)
	defer cleanup()

	payload := `{"scan_filter":{"q":"","page":1,"page_size":5},"validate_rules":["name_not_empty"],"clone":{"copies":2,"name_prefix":"clone","description_prefix":"copy"},"update_payload":{"description":"distributed","content":"## qa content"}}`
	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-quality",
		ToolScope:  "agent.template.quality_distribute",
		PayloadRef: payload,
		Metadata: map[string]any{
			"capability_id": (&templateQualityHandler{}).CapabilityID(),
			"session_id":    sessionID,
		},
	}

	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &resp))

	createdRaw, ok := resp["created_template_ids"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, createdRaw)

	updatedID, _ := resp["updated_template_id"].(string)
	require.NotEmpty(t, updatedID)
	updated, err := templateSvc.GetByID(ctx, mustParseUint64(t, updatedID))
	require.NoError(t, err)
	require.Equal(t, "## qa content", updated.Content)

	page, err := templateSvc.List(ctx, "", 1, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(page.List), len(createdRaw)+1)

	received := collectEvents(t, eventsCh, 3)
	require.Len(t, received, 3)
	require.Equal(t, "template.validate.completed", received[0].Type)
	require.Equal(t, "template.batch_clone.completed", received[1].Type)
	require.Equal(t, "template.update.completed", received[2].Type)
}

func TestCapabilityInvokerCRUDHandlers(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-crud")
	seed, err := templateSvc.Create(ctx, "SeedDoc", "init desc", "seed content")
	require.NoError(t, err)

	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "crud"), nil)

	listEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.list",
		PayloadRef: `{"q":"","page":1,"page_size":5}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.list",
		},
	}
	listResult, err := invoker.Invoke(context.Background(), listEnvelope)
	require.NoError(t, err)
	var listPayload map[string]any
	require.NoError(t, json.Unmarshal(listResult.Payload, &listPayload))
	items, ok := listPayload["list"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)

	readEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.read",
		PayloadRef: fmt.Sprintf(`{"template_id":%d}`, seed.ID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.read",
		},
	}
	readResult, err := invoker.Invoke(context.Background(), readEnvelope)
	require.NoError(t, err)
	var readPayload dbtemplate.Template
	require.NoError(t, json.Unmarshal(readResult.Payload, &readPayload))
	require.Equal(t, seed.ID, readPayload.ID)

	createEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.create",
		PayloadRef: `{"name":"MCP Template","description":"from crud test","content":"## crud"}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.create",
		},
	}
	createResult, err := invoker.Invoke(context.Background(), createEnvelope)
	require.NoError(t, err)
	var createPayload dbtemplate.Template
	require.NoError(t, json.Unmarshal(createResult.Payload, &createPayload))
	require.NotZero(t, createPayload.ID)

	updateEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.update",
		PayloadRef: fmt.Sprintf(`{"template_id":%d,"description":"updated via crud"}`, createPayload.ID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.update",
		},
	}
	updateResult, err := invoker.Invoke(context.Background(), updateEnvelope)
	require.NoError(t, err)
	var updatePayload dbtemplate.Template
	require.NoError(t, json.Unmarshal(updateResult.Payload, &updatePayload))
	require.Equal(t, "updated via crud", updatePayload.Description)
	require.Equal(t, "MCP Template", updatePayload.Name)

	deleteEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.delete",
		PayloadRef: fmt.Sprintf(`{"template_id":%d}`, createPayload.ID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.delete",
		},
	}
	deleteResult, err := invoker.Invoke(context.Background(), deleteEnvelope)
	require.NoError(t, err)
	var deletePayload map[string]any
	require.NoError(t, json.Unmarshal(deleteResult.Payload, &deletePayload))
	require.Equal(t, true, deletePayload["deleted"])

	_, err = templateSvc.GetByID(ctx, createPayload.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestCapabilityInvokerTemplatePrepareCollectsAndBuildsCapabilityRequest(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare"), nil)

	awaitingEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"create","template":{"title":"测试模板","description":"用于验证插件 CRUD"}}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	awaitingResult, err := invoker.Invoke(context.Background(), awaitingEnvelope)
	require.NoError(t, err)
	var awaiting map[string]any
	require.NoError(t, json.Unmarshal(awaitingResult.Payload, &awaiting))
	require.Equal(t, "awaiting_params", awaiting["status"])
	require.Equal(t, false, awaiting["ready_to_execute"])
	require.Equal(t, []interface{}{"template.content"}, awaiting["missing_fields"])

	readyEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"create","content":"这是一条测试内容","state":{"collected":{"action":"create","template":{"title":"测试模板","description":"用于验证插件 CRUD"}}}}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	readyResult, err := invoker.Invoke(context.Background(), readyEnvelope)
	require.NoError(t, err)
	var ready map[string]any
	require.NoError(t, json.Unmarshal(readyResult.Payload, &ready))
	require.Equal(t, "completed", ready["status"])
	require.Equal(t, true, ready["ready_to_execute"])
	request, ok := ready["capability_request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "com.powerx.plugins.base.template.create", request["capability_id"])
	reqPayload, ok := request["payload"].(map[string]any)
	require.True(t, ok)
	body, ok := reqPayload["body"].(map[string]any)
	require.True(t, ok)
	payload, ok := body["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "测试模板", payload["name"])
	require.Equal(t, "用于验证插件 CRUD", payload["description"])
	require.Equal(t, "这是一条测试内容", payload["content"])
}

func mustParseUint64(t *testing.T, value string) uint64 {
	t.Helper()
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	require.NoError(t, err)
	return parsed
}

func setupTemplateService(t *testing.T) (*gorm.DB, *srvtemplates.TemplateService) {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbtemplate.Template{}))
	return db, srvtemplates.NewTemplateService(db)
}

func collectEvents(t *testing.T, ch <-chan stream.Event, expected int) []stream.Event {
	t.Helper()
	received := make([]stream.Event, 0, expected)
	timeout := time.After(2 * time.Second)
	for len(received) < expected {
		select {
		case evt := <-ch:
			received = append(received, evt)
		case <-timeout:
			return received
		}
	}
	return received
}
