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

func TestCapabilityInvokerUsesOriginTenantForResourceOwnership(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "origin-tenant"), nil)

	originTenantUUID := "00000000-0000-0000-0000-000000000001"
	gatewayTenantUUID := "6b5d0240-9920-46da-b707-88200e0f51ea"
	envelope := &domain.IntegrationEnvelope{
		TenantUuid: gatewayTenantUUID,
		ToolScope:  "agent.template.create",
		PayloadRef: `{"name":"Origin Tenant Template","description":"owned by origin","content":"## origin"}`,
		Metadata: map[string]any{
			"capability_id":      "com.powerx.plugins.base.local.template.create",
			"origin_tenant_uuid": originTenantUUID,
		},
	}

	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.NotNil(t, result)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &payload))
	require.Contains(t, payload["content"], "已创建模板")
	links, ok := payload["links"].([]any)
	require.True(t, ok)
	require.Len(t, links, 1)

	originCtx := authx.ContextWithTenantUUID(context.Background(), originTenantUUID)
	originPage, err := templateSvc.List(originCtx, "Origin Tenant Template", 1, 10)
	require.NoError(t, err)
	require.Len(t, originPage.List, 1)
	require.Equal(t, originTenantUUID, originPage.List[0].TenantUuid)

	gatewayCtx := authx.ContextWithTenantUUID(context.Background(), gatewayTenantUUID)
	gatewayPage, err := templateSvc.List(gatewayCtx, "Origin Tenant Template", 1, 10)
	require.NoError(t, err)
	require.Empty(t, gatewayPage.List)
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
	items, ok := listPayload["items"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)
	require.Contains(t, listPayload["content"], "Seed")
	require.Contains(t, listPayload["content"], "/templates/crud?template_id=")
	pagination, ok := listPayload["pagination"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), pagination["page"])

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
	var readPayload map[string]any
	require.NoError(t, json.Unmarshal(readResult.Payload, &readPayload))
	readTemplate, ok := readPayload["template"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, fmt.Sprintf("%d", seed.ID), readTemplate["id"])

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
	var createPayload map[string]any
	require.NoError(t, json.Unmarshal(createResult.Payload, &createPayload))
	createdID := mustParseUint64(t, createPayload["id"].(string))
	require.NotZero(t, createdID)
	createdTemplate, ok := createPayload["template"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, createPayload["id"], createdTemplate["id"])

	updateEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.update",
		PayloadRef: fmt.Sprintf(`{"template_id":%d,"description":"updated via crud"}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.update",
		},
	}
	updateResult, err := invoker.Invoke(context.Background(), updateEnvelope)
	require.NoError(t, err)
	var updatePayload map[string]any
	require.NoError(t, json.Unmarshal(updateResult.Payload, &updatePayload))
	updatedTemplate, ok := updatePayload["template"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "updated via crud", updatedTemplate["description"])
	require.Equal(t, "MCP Template", updatedTemplate["name"])

	validateEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.validate",
		PayloadRef: fmt.Sprintf(`{"template_id":%d}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.validate",
		},
	}
	validateResult, err := invoker.Invoke(context.Background(), validateEnvelope)
	require.NoError(t, err)
	var validatePayload map[string]any
	require.NoError(t, json.Unmarshal(validateResult.Payload, &validatePayload))
	require.Equal(t, float64(createdID), validatePayload["template_id"])

	reviewEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.review",
		PayloadRef: fmt.Sprintf(`{"template_id":%d,"approved":true,"comments":"ok","reviewer":"qa"}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.review",
		},
	}
	reviewResult, err := invoker.Invoke(context.Background(), reviewEnvelope)
	require.NoError(t, err)
	var reviewPayload map[string]any
	require.NoError(t, json.Unmarshal(reviewResult.Payload, &reviewPayload))
	require.Equal(t, fmt.Sprintf("%d", createdID), reviewPayload["template_id"])
	require.Equal(t, "approved", reviewPayload["status"])

	publishEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.publish",
		PayloadRef: fmt.Sprintf(`{"template_id":%d,"channel":"tenant"}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.publish",
		},
	}
	publishResult, err := invoker.Invoke(context.Background(), publishEnvelope)
	require.NoError(t, err)
	var publishPayload map[string]any
	require.NoError(t, json.Unmarshal(publishResult.Payload, &publishPayload))
	require.Equal(t, fmt.Sprintf("%d", createdID), publishPayload["template_id"])
	require.Equal(t, "deployed", publishPayload["publish_status"])

	batchCloneEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.batch_clone",
		PayloadRef: fmt.Sprintf(`{"source_ids":[%d],"copies":1}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.batch_clone",
		},
	}
	batchCloneResult, err := invoker.Invoke(context.Background(), batchCloneEnvelope)
	require.NoError(t, err)
	var batchClonePayload map[string]any
	require.NoError(t, json.Unmarshal(batchCloneResult.Payload, &batchClonePayload))
	clonedIDs, ok := batchClonePayload["created_ids"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, clonedIDs)

	deleteEnvelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-crud",
		ToolScope:  "agent.template.delete",
		PayloadRef: fmt.Sprintf(`{"template_id":%d}`, createdID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.delete",
		},
	}
	deleteResult, err := invoker.Invoke(context.Background(), deleteEnvelope)
	require.NoError(t, err)
	var deletePayload map[string]any
	require.NoError(t, json.Unmarshal(deleteResult.Payload, &deletePayload))
	require.Equal(t, true, deletePayload["deleted"])
	require.Equal(t, fmt.Sprintf("%d", createdID), deletePayload["id"])

	_, err = templateSvc.GetByID(ctx, createdID)
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
	require.Equal(t, "awaiting_params", awaitingResult.Status)
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
	require.Equal(t, "completed", readyResult.Status)
	var ready map[string]any
	require.NoError(t, json.Unmarshal(readyResult.Payload, &ready))
	require.Equal(t, "completed", ready["status"])
	require.Equal(t, true, ready["ready_to_execute"])
	request, ok := ready["capability_request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "com.powerx.plugins.base.template.create", request["capability_id"])
	reqPayload, ok := request["payload"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, reqPayload, "endpoint")
	require.NotContains(t, reqPayload, "body")
	require.Equal(t, "测试模板", reqPayload["name"])
	require.Equal(t, "用于验证插件 CRUD", reqPayload["description"])
	require.Equal(t, "这是一条测试内容", reqPayload["content"])
}

func TestCapabilityInvokerTemplatePrepareAcceptsLocalCapabilityID(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-local"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-local",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"create","template":{"title":"测试模板","description":"用于验证插件 CRUD","content":"这是一条测试内容"}}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.local.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	var ready map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &ready))
	require.Equal(t, "completed", ready["status"])
	require.Equal(t, true, ready["ready_to_execute"])
	request, ok := ready["capability_request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "com.powerx.plugins.base.local.template.create", request["capability_id"])
}

func TestCapabilityInvokerTemplatePreparePassesListQueryAndPagination(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-list"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-list",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"list","q":"合同","page":2,"page_size":20}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	var ready map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &ready))
	request, ok := ready["capability_request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "com.powerx.plugins.base.template.list", request["capability_id"])
	reqPayload, ok := request["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "list", reqPayload["action"])
	require.Equal(t, "合同", reqPayload["q"])
	require.Equal(t, float64(2), reqPayload["page"])
	require.Equal(t, float64(20), reqPayload["page_size"])
}

func TestCapabilityInvokerTemplatePrepareAcceptsStringNumericTemplateID(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-template-id-string"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-id-string",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"delete","template_id":"123"}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "awaiting_params", result.Status)

	var awaiting map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &awaiting))
	require.Equal(t, false, awaiting["ready_to_execute"])
	require.Equal(t, []interface{}{"confirmation"}, awaiting["missing_fields"])
	statePatch, ok := awaiting["state_patch"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(123), statePatch["template_id"])
}

func TestCapabilityInvokerTemplatePrepareResolvesTemplateNameAndRequiresDeleteConfirmation(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-prepare-id-name")
	seed, err := templateSvc.Create(ctx, "测试模板", "用于名称解析", "content")
	require.NoError(t, err)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-template-id-name"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-id-name",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"delete","template_id":"测试模板"}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "awaiting_params", result.Status)

	var awaiting map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &awaiting))
	require.Equal(t, false, awaiting["ready_to_execute"])
	require.Equal(t, []interface{}{"confirmation"}, awaiting["missing_fields"])
	require.Contains(t, awaiting["message"], "查看模板详情")
	require.Contains(t, awaiting["message"], fmt.Sprintf("template_id=%d", seed.ID))
	statePatch, ok := awaiting["state_patch"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(seed.ID), statePatch["template_id"])
	require.Equal(t, "测试模板", statePatch["template_name"])
}

func TestCapabilityInvokerTemplatePrepareExecutesDeleteAfterConfirmation(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-prepare-confirm-delete")
	seed, err := templateSvc.Create(ctx, "测试模板", "用于确认删除", "content")
	require.NoError(t, err)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-confirm-delete"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-confirm-delete",
		ToolScope:  "agent.template.prepare",
		PayloadRef: fmt.Sprintf(`{"action":"delete","user_message":"确认删除","state":{"collected":{"action":"delete","template_id":%d,"template_name":"测试模板"}}}`, seed.ID),
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	var ready map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &ready))
	require.Equal(t, true, ready["ready_to_execute"])
	request, ok := ready["capability_request"].(map[string]any)
	require.True(t, ok)
	reqPayload, ok := request["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(seed.ID), reqPayload["template_id"])
}

func TestCapabilityInvokerTemplatePrepareReturnsDuplicateTemplateCandidates(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-prepare-duplicate")
	first, err := templateSvc.Create(ctx, "重名模板", "first", "content")
	require.NoError(t, err)
	second, err := templateSvc.Create(ctx, "重名模板", "second", "content")
	require.NoError(t, err)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-duplicate"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-duplicate",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"delete","template_name":"重名模板"}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "awaiting_params", result.Status)

	var awaiting map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &awaiting))
	require.Equal(t, false, awaiting["ready_to_execute"])
	require.Equal(t, []interface{}{"template_ref"}, awaiting["missing_fields"])
	require.Contains(t, awaiting["message"], fmt.Sprintf("template_id=%d", first.ID))
	require.Contains(t, awaiting["message"], fmt.Sprintf("template_id=%d", second.ID))
	require.Contains(t, awaiting["message"], "模板 ID")
	statePatch, ok := awaiting["state_patch"].(map[string]any)
	require.True(t, ok)
	candidates, ok := statePatch["template_candidates"].([]interface{})
	require.True(t, ok)
	require.Len(t, candidates, 2)
}

func TestCapabilityInvokerTemplatePrepareAsksForTemplateNameWhenLookupMisses(t *testing.T) {
	_, templateSvc := setupTemplateService(t)
	invoker := NewCapabilityInvoker(templateSvc, stream.NewBroker(), logrus.New().WithField("test", "prepare-template-ref-miss"), nil)

	envelope := &domain.IntegrationEnvelope{
		TenantUuid: "tenant-prepare-ref-miss",
		ToolScope:  "agent.template.prepare",
		PayloadRef: `{"action":"delete","template_id":"不存在的模板"}`,
		Metadata: map[string]any{
			"capability_id": "com.powerx.plugins.base.template.prepare",
		},
	}
	result, err := invoker.Invoke(context.Background(), envelope)
	require.NoError(t, err)
	require.Equal(t, "awaiting_params", result.Status)

	var awaiting map[string]any
	require.NoError(t, json.Unmarshal(result.Payload, &awaiting))
	require.Equal(t, false, awaiting["ready_to_execute"])
	require.Equal(t, []interface{}{"template_ref"}, awaiting["missing_fields"])
	require.NotContains(t, awaiting["message"], "ID")
}

func TestLocalizeCapabilityIDForRequestIsPluginAgnostic(t *testing.T) {
	envelope := &domain.IntegrationEnvelope{
		Metadata: map[string]any{
			"capability_id": "com.example.plugins.demo.local.template.prepare",
		},
	}

	got := localizeCapabilityIDForRequest("com.example.plugins.demo.template.create", envelope)
	require.Equal(t, "com.example.plugins.demo.local.template.create", got)
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
