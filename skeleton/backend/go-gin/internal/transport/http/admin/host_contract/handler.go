package host_contract

import (
	"encoding/json"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	fwmodule "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	fwcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/capability"
	fwintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/integration"
	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	fwmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	fwnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	fwpluginruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/pluginruntime"
	powerxagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	powerxintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	powerxmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	cache         *cache.Runtime
	tasks         *taskcenter.Runtime
	directory     fwiamcontracts.DirectoryService
	authorizer    fwiamcontracts.AuthzService
	knowledge     fwknowledge.KnowledgeProvider
	agent         *fwagent.Runtime
	ai            *fwai.Runtime
	capabilities  *fwcapability.Runtime
	media         *fwmedia.Runtime
	integration   *fwintegration.Runtime
	skills        *fwskills.Runtime
	notifications *fwnotifications.Runtime
	pluginRuntime *fwpluginruntime.Runtime
	realtime      []frameworkrealtime.Descriptor
	providerMode  fwprovider.Mode
}

func NewHandler(deps *app.Deps) *Handler {
	h := &Handler{}
	if deps != nil {
		h.cache = deps.CacheRuntime
		h.tasks = deps.TaskCenterRuntime
		h.directory = deps.IAMDirectoryService
		h.authorizer = deps.IAMAuthzService
		h.knowledge = deps.KnowledgeProvider
		h.agent = deps.AgentLifecycle
		h.ai = deps.AIInvocation
		h.capabilities = deps.CapabilityAccess
		h.media = deps.MediaCatalog
		h.integration = deps.IntegrationGateway
		h.skills = deps.SkillInvocation
		h.notifications = deps.NotificationDelivery
		h.pluginRuntime = deps.PluginRuntime
		h.realtime = append([]frameworkrealtime.Descriptor(nil), deps.RealtimeDescriptors...)
		h.providerMode = deps.ProviderMode
	}
	return h
}

func (h *Handler) Probe(c *gin.Context) {
	var request hostcontract.ProbeRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		respondError(c, http.StatusBadRequest, hostcontract.ReasonInvalidArgument)
		return
	}
	tenantUUID, ok := authx.TenantUUIDFromContext(c.Request.Context())
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		respondError(c, http.StatusUnauthorized, hostcontract.ReasonUnauthorized)
		return
	}
	if hasTenantOverride(c, tenantUUID) {
		respondError(c, http.StatusBadRequest, hostcontract.ReasonTenantOverrideForbidden)
		return
	}
	if err := hostcontract.ValidateNoTenantOverride(request.Input); err != nil {
		respondError(c, http.StatusBadRequest, hostcontract.CodeOf(err))
		return
	}
	descriptor, err := hostcontract.LookupOperation(request.Module, request.Operation)
	if err != nil {
		respondError(c, http.StatusBadRequest, hostcontract.CodeOf(err))
		return
	}
	if descriptor.RequiresConfirmation && !request.Confirm {
		respondError(c, http.StatusConflict, hostcontract.ReasonConfirmationRequired)
		return
	}
	result := hostcontract.ProbeResult{
		Module:       descriptor.Module,
		Operation:    descriptor.Operation,
		ProviderMode: providerMode(h.providerMode),
		CapabilityID: descriptor.CapabilityID,
		ObservedAt:   time.Now().UTC(),
	}
	if descriptor.Operation == hostcontract.OperationStatus {
		if err := h.probeBinding(request.Input, &result); err != nil {
			status, reason, trace := mapTypedClientError(err)
			result.TraceID = trace
			respondProbeError(c, result, status, reason)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
		return
	}
	switch descriptor.Module {
	case hostcontract.ModuleCache, hostcontract.ModuleTaskCenter:
		if err := h.probeStorage(c, tenantUUID, request.Input, &result); err != nil {
			status, reason, trace := mapTypedClientError(err)
			result.TraceID = trace
			respondProbeError(c, result, status, reason)
			return
		}
	case hostcontract.ModuleIAM:
		if h.directory == nil {
			respondProbeError(c, result, http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency)
			return
		}
		if err := h.probeIAM(c, tenantUUID, request.Input, &result); err != nil {
			status, reason := mapIAMError(err)
			respondProbeError(c, result, status, reason)
			return
		}
	case hostcontract.ModuleKnowledge:
		if h.knowledge == nil {
			respondProbeError(c, result, http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency)
			return
		}
		if err := h.probeKnowledge(c, tenantUUID, request.Input, &result); err != nil {
			status, reason, traceID := mapTypedClientError(err)
			result.TraceID = firstNonEmpty(traceID, result.TraceID)
			respondProbeError(c, result, status, reason)
			return
		}
	case hostcontract.ModuleMedia, hostcontract.ModuleAgent, hostcontract.ModuleAI,
		hostcontract.ModuleCapabilityRegistry, hostcontract.ModuleIntegrationGateway,
		hostcontract.ModuleSkills, hostcontract.ModuleNotifications, hostcontract.ModulePluginRuntime:
		if err := h.probeRuntimeModule(c, request.Input, &result); err != nil {
			status, reason, traceID := mapTypedClientError(err)
			result.TraceID = firstNonEmpty(traceID, result.TraceID)
			respondProbeError(c, result, status, reason)
			return
		}
	default:
		respondProbeError(c, result, http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// probeRuntimeModule only calls an exported Framework client method. It never
// reconstructs a Core URL, falls back to a local store, or accepts a tenant
// from the request. Command-only contracts deliberately expose binding status
// instead of issuing a side-effecting request just to prove connectivity.
func (h *Handler) probeRuntimeModule(c *gin.Context, input map[string]any, result *hostcontract.ProbeResult) error {
	if result == nil {
		return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
	}
	ctx := c.Request.Context()
	switch result.Module {
	case hostcontract.ModuleMedia:
		if result.Operation != hostcontract.OperationStatus && result.Operation != hostcontract.OperationMediaAssetsList {
			return h.probeMediaTransfer(c, input, result)
		}
		if h.media == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "read_operations": []string{string(hostcontract.OperationMediaAssetsList)}}
		case hostcontract.OperationMediaAssetsList:
			if err := ensureInputKeys(input, "page", "page_size", "keyword"); err != nil {
				return err
			}
			page, err := inputPositiveInt(input, "page", 1, 10000)
			if err != nil {
				return err
			}
			pageSize, err := inputPositiveInt(input, "page_size", 20, 100)
			if err != nil {
				return err
			}
			keyword, err := inputOptionalString(input, "keyword")
			if err != nil {
				return err
			}
			catalog, err := h.media.Assets()
			if err != nil {
				return err
			}
			assets, err := catalog.ListAssets(ctx, powerxmedia.ListAssetsInput{Page: page, PageSize: pageSize, Keyword: keyword})
			if err != nil {
				return err
			}
			result.TraceID = assets.TraceID
			result.Result = map[string]any{"assets": assets}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleAgent:
		if strings.HasPrefix(string(result.Operation), "session.") || result.Operation == "sessions.list" {
			return h.probeAgentSession(c, input, result)
		}
		if h.agent == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		lifecycle, err := h.agent.Lifecycle()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "binding": "runtime"}
		case hostcontract.OperationAgentHealthSummary:
			if err := ensureInputKeys(input, "agent_uuid"); err != nil {
				return err
			}
			agentUUID, err := inputUUID(input, "agent_uuid")
			if err != nil {
				return err
			}
			summary, err := lifecycle.GetHealthSummary(ctx, agentUUID)
			if err != nil {
				return err
			}
			result.Result = map[string]any{"health": summary}
		case hostcontract.OperationAgentFreeze, hostcontract.OperationAgentRecover:
			if err := ensureInputKeys(input, "agent_uuid", "reason"); err != nil {
				return err
			}
			agentUUID, err := inputUUID(input, "agent_uuid")
			if err != nil {
				return err
			}
			reason, err := inputOptionalString(input, "reason")
			if err != nil {
				return err
			}
			control := powerxagent.BridgeControlInput{Reason: reason, TraceID: strings.TrimSpace(c.GetString("request_id"))}
			var output *powerxagent.BridgeLifecycleResult
			if result.Operation == hostcontract.OperationAgentFreeze {
				output, err = lifecycle.Freeze(ctx, agentUUID, control)
			} else {
				output, err = lifecycle.Recover(ctx, agentUUID, control)
			}
			if err != nil {
				return err
			}
			result.Result = map[string]any{"lifecycle": output}
		case hostcontract.OperationAgentRebalance:
			if err := ensureInputKeys(input, "agent_uuid", "target_capacity_instances", "reason"); err != nil {
				return err
			}
			agentUUID, err := inputUUID(input, "agent_uuid")
			if err != nil {
				return err
			}
			capacity, err := inputPositiveInt(input, "target_capacity_instances", 0, 100000)
			if err != nil || capacity == 0 {
				return &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
			}
			reason, err := inputOptionalString(input, "reason")
			if err != nil {
				return err
			}
			output, err := lifecycle.Rebalance(ctx, agentUUID, powerxagent.BridgeRebalanceInput{TargetCapacityInstances: int32(capacity), Reason: reason, TraceID: strings.TrimSpace(c.GetString("request_id"))})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"lifecycle": output}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleAI:
		if result.Operation == "llm.invoke" || result.Operation == "embedding.invoke" {
			return h.probeAIExecution(c, input, result)
		}
		if h.ai == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "read_operations": []string{string(hostcontract.OperationAIModelsList)}}
		case hostcontract.OperationAIModelsList:
			if err := ensureInputKeys(input, "provider"); err != nil {
				return err
			}
			provider, err := inputOptionalString(input, "provider")
			if err != nil {
				return err
			}
			generative, err := h.ai.Generative()
			if err != nil {
				return err
			}
			models, err := generative.ListLLMModels(ctx, provider)
			if err != nil {
				return err
			}
			result.Result = map[string]any{"models": models}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleCapabilityRegistry:
		if h.capabilities == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		registry, err := h.capabilities.Registry()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "read_operations": []string{string(hostcontract.OperationCapabilityRegistryList)}}
		case hostcontract.OperationCapabilityRegistryList:
			if err := ensureInputKeys(input, "page", "page_size", "plugin_id", "source"); err != nil {
				return err
			}
			page, err := inputPositiveInt(input, "page", 1, 10000)
			if err != nil {
				return err
			}
			pageSize, err := inputPositiveInt(input, "page_size", 20, 100)
			if err != nil {
				return err
			}
			pluginID, err := inputOptionalString(input, "plugin_id")
			if err != nil {
				return err
			}
			source, err := inputOptionalString(input, "source")
			if err != nil {
				return err
			}
			items, err := registry.List(ctx, powerxcapability.ListInput{Page: page, PageSize: pageSize, PluginID: pluginID, Source: source})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"items": items, "page": page, "page_size": pageSize}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleIntegrationGateway:
		if h.integration == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		gateway, err := h.integration.Gateway()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "binding": "runtime"}
		case hostcontract.OperationIntegrationRoutesList:
			if err := ensureInputKeys(input, "capability_id", "channel"); err != nil {
				return err
			}
			capabilityID, err := inputOptionalString(input, "capability_id")
			if err != nil {
				return err
			}
			channel, err := inputOptionalString(input, "channel")
			if err != nil {
				return err
			}
			items, err := gateway.ListRoutes(ctx, powerxintegration.ListRoutesInput{CapabilityID: capabilityID, Channel: channel})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"items": items}
		case hostcontract.OperationIntegrationRouteInvoke:
			if err := ensureInputKeys(input, "route_slug", "payload", "context", "idempotency_key"); err != nil {
				return err
			}
			routeSlug, err := inputRequiredString(input, "route_slug")
			if err != nil {
				return err
			}
			payload, err := inputOptionalObject(input, "payload")
			if err != nil {
				return err
			}
			if payload == nil {
				return &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
			}
			callContext, err := inputOptionalObject(input, "context")
			if err != nil {
				return err
			}
			idempotencyKey, err := inputOptionalString(input, "idempotency_key")
			if err != nil {
				return err
			}
			output, err := gateway.InvokeRoute(ctx, routeSlug, powerxintegration.InvokeRouteInput{Payload: payload, Context: callContext, IdempotencyKey: idempotencyKey})
			if err != nil {
				return err
			}
			if output != nil {
				result.TraceID = output.TraceID
			}
			result.Result = map[string]any{"invocation": output}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleSkills:
		if h.skills == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		invoker, err := h.skills.Invoker()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "binding": "runtime", "read_operations": []string{}}
		case hostcontract.OperationSkillInvoke:
			if err := ensureInputKeys(input, "skill_id", "version", "payload", "context"); err != nil {
				return err
			}
			skillID, err := inputRequiredString(input, "skill_id")
			if err != nil {
				return err
			}
			version, err := inputOptionalString(input, "version")
			if err != nil {
				return err
			}
			payload, err := inputOptionalObject(input, "payload")
			if err != nil {
				return err
			}
			invocationContext, err := inputOptionalObject(input, "context")
			if err != nil {
				return err
			}
			output, err := invoker.Invoke(ctx, powerxskills.InvokeInput{SkillID: skillID, Version: version, Payload: payload, Context: invocationContext})
			if err != nil {
				return err
			}
			if output != nil {
				result.TraceID = output.TraceID
			}
			result.Result = map[string]any{"invocation": output}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModuleNotifications:
		if h.notifications == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		publisher, err := h.notifications.Publisher()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "binding": "runtime", "read_operations": []string{}}
		case hostcontract.OperationNotificationCreate:
			if err := ensureInputKeys(input, "title", "content", "type", "category", "is_important", "member_uuid", "metadata"); err != nil {
				return err
			}
			title, err := inputRequiredString(input, "title")
			if err != nil {
				return err
			}
			content, err := inputRequiredString(input, "content")
			if err != nil {
				return err
			}
			typeName, err := inputOptionalString(input, "type")
			if err != nil {
				return err
			}
			category, err := inputOptionalString(input, "category")
			if err != nil {
				return err
			}
			important, err := inputOptionalBool(input, "is_important")
			if err != nil {
				return err
			}
			memberUUID, err := inputOptionalUUID(input, "member_uuid")
			if err != nil {
				return err
			}
			metadata, err := inputOptionalObject(input, "metadata")
			if err != nil {
				return err
			}
			notification, err := publisher.Create(ctx, powerxnotifications.CreateInput{Title: title, Content: content, Type: typeName, Category: category, IsImportant: important, MemberUUID: memberUUID, Metadata: metadata})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"notification": notification}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	case hostcontract.ModulePluginRuntime:
		if h.pluginRuntime == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		service, err := h.pluginRuntime.Service()
		if err != nil {
			return err
		}
		switch result.Operation {
		case hostcontract.OperationStatus:
			if err := ensureInputKeys(input); err != nil {
				return err
			}
			result.Result = map[string]any{"ready": true, "realtime_descriptor_count": len(h.realtime), "realtime_descriptors": h.realtime}
		case hostcontract.OperationPluginRuntimeKnowledgeSpacesList:
			if err := ensureInputKeys(input, "page", "page_size", "status", "keyword"); err != nil {
				return err
			}
			page, err := inputPositiveInt(input, "page", 1, 10000)
			if err != nil {
				return err
			}
			pageSize, err := inputPositiveInt(input, "page_size", 20, 100)
			if err != nil {
				return err
			}
			status, err := inputOptionalString(input, "status")
			if err != nil {
				return err
			}
			keyword, err := inputOptionalString(input, "keyword")
			if err != nil {
				return err
			}
			spaces, err := service.ListKnowledgeSpaces(ctx, powerxruntime.ListKnowledgeSpacesInput{Page: page, PageSize: pageSize, Status: status, Keyword: keyword})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"spaces": spaces}
		case hostcontract.OperationPluginRuntimeAgentsList:
			if err := ensureInputKeys(input, "env", "status"); err != nil {
				return err
			}
			env, err := inputOptionalString(input, "env")
			if err != nil {
				return err
			}
			status, err := inputOptionalString(input, "status")
			if err != nil {
				return err
			}
			agents, err := service.ListAgents(ctx, env, status)
			if err != nil {
				return err
			}
			result.Result = map[string]any{"agents": agents}
		case hostcontract.OperationPluginRuntimeAgentInstantiate:
			if err := ensureInputKeys(input, "name", "env", "key", "description", "skill_uuids", "knowledge_space_uuids", "parameters", "metadata"); err != nil {
				return err
			}
			name, err := inputRequiredString(input, "name")
			if err != nil {
				return err
			}
			env, err := inputOptionalString(input, "env")
			if err != nil {
				return err
			}
			key, err := inputOptionalString(input, "key")
			if err != nil {
				return err
			}
			description, err := inputOptionalString(input, "description")
			if err != nil {
				return err
			}
			skillIDs, err := inputUUIDsOptional(input, "skill_uuids")
			if err != nil {
				return err
			}
			knowledgeIDs, err := inputUUIDsOptional(input, "knowledge_space_uuids")
			if err != nil {
				return err
			}
			parameters, err := inputOptionalObject(input, "parameters")
			if err != nil {
				return err
			}
			metadata, err := inputOptionalObject(input, "metadata")
			if err != nil {
				return err
			}
			agent, err := service.InstantiateAgent(ctx, powerxruntime.InstantiateAgentInput{Name: name, Environment: env, Key: key, Description: description, SkillIDs: skillIDs, KnowledgeBaseIDs: knowledgeIDs, Parameters: parameters, Meta: metadata})
			if err != nil {
				return err
			}
			result.Result = map[string]any{"agent": agent}
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	default:
		return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
	}
	return nil
}

func (h *Handler) probeKnowledge(c *gin.Context, tenant string, input map[string]any, result *hostcontract.ProbeResult) error {
	traceID := strings.TrimSpace(c.GetString("request_id"))
	switch result.Operation {
	case hostcontract.OperationKnowledgeSpacesList:
		if err := ensureInputKeys(input); err != nil {
			return err
		}
		items, err := h.knowledge.ListSpaces(c.Request.Context(), fwknowledge.ListSpacesInput{TenantUUID: tenant})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"items": items}
	case hostcontract.OperationKnowledgeSearch:
		if err := ensureInputKeys(input, "query", "space_uuids", "limit"); err != nil {
			return err
		}
		query, err := inputRequiredString(input, "query")
		if err != nil {
			return err
		}
		spaces, err := inputUUIDsOptional(input, "space_uuids")
		if err != nil {
			return err
		}
		limit, err := inputPositiveInt(input, "limit", 10, 100)
		if err != nil {
			return err
		}
		item, err := h.knowledge.Search(c.Request.Context(), fwknowledge.KnowledgeQuery{TenantUUID: tenant, Query: query, SpaceIDs: spaces, Limit: limit, TraceID: traceID})
		if err != nil {
			return err
		}
		result.TraceID = item.TraceID
		result.Result = map[string]any{"search": item}
	case hostcontract.OperationKnowledgeIndexJobGet:
		if err := ensureInputKeys(input, "job_uuid"); err != nil {
			return err
		}
		jobID, err := inputUUID(input, "job_uuid")
		if err != nil {
			return err
		}
		item, err := h.knowledge.GetIndexJob(c.Request.Context(), fwknowledge.IndexJobQuery{TenantUUID: tenant, JobID: jobID, TraceID: traceID})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"job": item}
	case hostcontract.OperationKnowledgeDocumentCreate:
		if err := ensureInputKeys(input, "space_uuid", "title", "uri", "content", "content_type", "checksum", "version", "tags"); err != nil {
			return err
		}
		spaceID, err := inputUUID(input, "space_uuid")
		if err != nil {
			return err
		}
		title, err := inputRequiredString(input, "title")
		if err != nil {
			return err
		}
		uri, err := inputRequiredString(input, "uri")
		if err != nil {
			return err
		}
		content, err := inputRequiredString(input, "content")
		if err != nil {
			return err
		}
		contentType, err := inputRequiredString(input, "content_type")
		if err != nil {
			return err
		}
		checksum, err := inputRequiredString(input, "checksum")
		if err != nil {
			return err
		}
		version, err := inputRequiredString(input, "version")
		if err != nil {
			return err
		}
		tags, err := inputStringsOptional(input, "tags")
		if err != nil {
			return err
		}
		item, err := h.knowledge.UpsertDocument(c.Request.Context(), fwknowledge.KnowledgeDocument{TenantUUID: tenant, SpaceID: spaceID, Title: title, URI: uri, Content: content, ContentType: contentType, Checksum: checksum, Version: version, Tags: tags})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"job": item}
	case hostcontract.OperationKnowledgeDocumentDelete:
		if err := ensureInputKeys(input, "space_uuid", "document_uuid"); err != nil {
			return err
		}
		spaceID, err := inputUUID(input, "space_uuid")
		if err != nil {
			return err
		}
		documentID, err := inputUUID(input, "document_uuid")
		if err != nil {
			return err
		}
		item, err := h.knowledge.DeleteDocument(c.Request.Context(), fwknowledge.DeleteDocumentInput{TenantUUID: tenant, SpaceID: spaceID, DocumentID: documentID, TraceID: traceID})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"job": item}
	case hostcontract.OperationKnowledgeIndexRebuild:
		if err := ensureInputKeys(input, "space_uuid"); err != nil {
			return err
		}
		spaceID, err := inputUUID(input, "space_uuid")
		if err != nil {
			return err
		}
		item, err := h.knowledge.Reindex(c.Request.Context(), fwknowledge.ReindexInput{TenantUUID: tenant, SpaceID: spaceID, TraceID: traceID})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"job": item}
	default:
		return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
	}
	return nil
}

func (h *Handler) probeIAM(c *gin.Context, tenantUUID string, input map[string]any, result *hostcontract.ProbeResult) error {
	if result == nil {
		return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
	}
	switch result.Operation {
	case hostcontract.OperationIAMTenant:
		if err := ensureInputKeys(input); err != nil {
			return err
		}
		item, err := h.directory.GetTenant(c.Request.Context(), tenantUUID)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"tenant": item}
	case hostcontract.OperationIAMMembersList:
		if err := ensureInputKeys(input, "page", "page_size"); err != nil {
			return err
		}
		page, err := inputPositiveInt(input, "page", 1, 10000)
		if err != nil {
			return err
		}
		pageSize, err := inputPositiveInt(input, "page_size", 20, 100)
		if err != nil {
			return err
		}
		items, err := h.directory.ListMembersPage(c.Request.Context(), tenantUUID, fwiamcontracts.MemberPageRequest{Page: page, PageSize: pageSize})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"page": items}
	case hostcontract.OperationIAMMemberGet:
		if err := ensureInputKeys(input, "member_uuid"); err != nil {
			return err
		}
		memberUUID, err := inputUUID(input, "member_uuid")
		if err != nil {
			return err
		}
		item, err := h.directory.GetMember(c.Request.Context(), tenantUUID, memberUUID)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"member": item}
	case hostcontract.OperationIAMMembersBatchGet:
		if err := ensureInputKeys(input, "member_uuids"); err != nil {
			return err
		}
		memberUUIDs, err := inputUUIDs(input, "member_uuids")
		if err != nil {
			return err
		}
		items, err := h.directory.BatchGetMembers(c.Request.Context(), tenantUUID, memberUUIDs)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"items": items}
	case hostcontract.OperationIAMMembersBatchResolve:
		if err := ensureInputKeys(input, "member_uuids"); err != nil {
			return err
		}
		memberUUIDs, err := inputUUIDs(input, "member_uuids")
		if err != nil {
			return err
		}
		items, err := h.directory.BatchResolveMembers(c.Request.Context(), tenantUUID, memberUUIDs)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"resolution": items}
	case hostcontract.OperationIAMMembersResolveDisplayNames:
		if err := ensureInputKeys(input, "display_names"); err != nil {
			return err
		}
		displayNames, err := inputStrings(input, "display_names")
		if err != nil {
			return err
		}
		items, err := h.directory.BatchResolveMembersByDisplayNames(c.Request.Context(), tenantUUID, displayNames)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"resolution": items}
	case hostcontract.OperationIAMDepartmentsList:
		if err := ensureInputKeys(input); err != nil {
			return err
		}
		items, err := h.directory.ListDepartments(c.Request.Context(), tenantUUID)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"items": items}
	case hostcontract.OperationIAMRolesList:
		if err := ensureInputKeys(input); err != nil {
			return err
		}
		items, err := h.directory.ListRoles(c.Request.Context(), tenantUUID)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"items": items}
	case hostcontract.OperationIAMPermissionsList:
		if err := ensureInputKeys(input); err != nil {
			return err
		}
		items, err := h.directory.ListPermissions(c.Request.Context(), tenantUUID)
		if err != nil {
			return err
		}
		result.Result = map[string]any{"items": items}
	case hostcontract.OperationIAMAuthorizationCheck:
		if h.authorizer == nil {
			return &hostcontract.Error{Reason: hostcontract.ReasonUpstreamDependency}
		}
		if err := ensureInputKeys(input, "member_uuid", "user_uuid", "resource", "action"); err != nil {
			return err
		}
		memberUUID, err := inputOptionalUUID(input, "member_uuid")
		if err != nil {
			return err
		}
		userUUID, err := inputOptionalUUID(input, "user_uuid")
		if err != nil {
			return err
		}
		resource, err := inputRequiredString(input, "resource")
		if err != nil {
			return err
		}
		action, err := inputRequiredString(input, "action")
		if err != nil {
			return err
		}
		decision, err := h.authorizer.Authorize(c.Request.Context(), fwiamcontracts.AuthorizationRequest{TenantUUID: tenantUUID, MemberUUID: memberUUID, UserUUID: userUUID, Resource: resource, Action: action, TraceID: strings.TrimSpace(c.GetString("request_id"))})
		if err != nil {
			return err
		}
		result.Result = map[string]any{"decision": decision}
		if decision != nil {
			result.TraceID = decision.TraceID
			result.ReasonCode = hostcontract.ReasonCode(decision.ReasonCode)
		}
	default:
		return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
	}
	return nil
}

func hasTenantOverride(c *gin.Context, tenant string) bool {
	for _, key := range []string{"tenant_uuid", "tenantUuid", "tenant_id", "tenantId"} {
		if c.Request.URL.Query().Has(key) {
			return true
		}
	}
	// Browser SDK tenant hints may confirm, but never establish, authority.
	for _, key := range []string{"tenant_uuid", "X-Tenant-UUID"} {
		values := c.Request.Header.Values(key)
		if len(values) > 1 {
			return true
		}
		if len(values) == 1 && values[0] != tenant {
			return true
		}
	}
	return false
}

func providerMode(mode fwprovider.Mode) string {
	if strings.TrimSpace(mode.String()) == "" {
		return string(fwprovider.ModeLocal)
	}
	return mode.String()
}

func respondError(c *gin.Context, status int, reason hostcontract.ReasonCode) {
	c.JSON(status, gin.H{"success": false, "error": hostcontract.ErrorEnvelope{ErrorCode: string(reason), ReasonCode: reason, TraceID: strings.TrimSpace(c.GetString("request_id"))}})
}

func respondProbeError(c *gin.Context, result hostcontract.ProbeResult, status int, reason hostcontract.ReasonCode) {
	result.ReasonCode = reason
	c.JSON(status, gin.H{"success": false, "data": result, "error": hostcontract.ErrorEnvelope{ErrorCode: string(reason), ReasonCode: reason, TraceID: result.TraceID}})
}

func mapIAMError(err error) (int, hostcontract.ReasonCode) {
	if code := fwiamerrors.CodeOf(err); code != "" {
		return fwiamerrors.StatusCode(err), hostcontract.ReasonCode(code)
	}
	return http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency
}

func mapTypedClientError(err error) (int, hostcontract.ReasonCode, string) {
	var bindingError *fwmodule.Error
	if errors.As(err, &bindingError) {
		return http.StatusServiceUnavailable, hostcontract.ReasonCode(bindingError.Code), ""
	}
	var upstream *hostapi.HTTPError
	if errors.As(err, &upstream) {
		return upstream.StatusCode, hostcontract.ReasonCode(upstream.ReasonCode), upstream.RequestID
	}
	for _, invalid := range []error{cache.ErrInvalidArgument, taskcenter.ErrInvalidArgument} {
		if errors.Is(err, invalid) {
			return 400, hostcontract.ReasonCode(invalid.Error()), ""
		}
	}
	if errors.Is(err, taskcenter.ErrNotFound) {
		return 404, hostcontract.ReasonCode(taskcenter.ErrNotFound.Error()), ""
	}
	if errors.Is(err, taskcenter.ErrConflict) {
		return 409, hostcontract.ReasonCode(taskcenter.ErrConflict.Error()), ""
	}
	if err == nil {
		return http.StatusOK, "", ""
	}
	if code := hostcontract.CodeOf(err); code != hostcontract.ReasonUpstreamDependency {
		return http.StatusBadRequest, code, ""
	}
	var agentError *powerxagent.Error
	if errors.As(err, &agentError) && agentError != nil {
		switch agentError.Code {
		case powerxagent.ErrCodeUnauthorized:
			return http.StatusUnauthorized, hostcontract.ReasonUnauthorized, agentError.TraceID
		case powerxagent.ErrCodeForbidden:
			return http.StatusForbidden, hostcontract.ReasonForbidden, agentError.TraceID
		case powerxagent.ErrCodeNotFound:
			return http.StatusNotFound, hostcontract.ReasonUpstreamDependency, agentError.TraceID
		default:
			return http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency, agentError.TraceID
		}
	}
	var capabilityError *powerxcapability.HTTPError
	if errors.As(err, &capabilityError) && capabilityError != nil {
		return mapHTTPStatus(capabilityError.StatusCode), reasonForHTTPStatus(capabilityError.StatusCode), ""
	}
	var aiError *powerxai.HTTPError
	if errors.As(err, &aiError) && aiError != nil {
		return mapHTTPStatus(aiError.StatusCode), reasonForHTTPStatus(aiError.StatusCode), ""
	}
	var skillError *powerxskills.HTTPError
	if errors.As(err, &skillError) && skillError != nil {
		return mapHTTPStatus(skillError.StatusCode), reasonForHTTPStatus(skillError.StatusCode), ""
	}
	var notificationError *powerxnotifications.HTTPError
	if errors.As(err, &notificationError) && notificationError != nil {
		return mapHTTPStatus(notificationError.StatusCode), reasonForHTTPStatus(notificationError.StatusCode), ""
	}
	return http.StatusFailedDependency, hostcontract.ReasonUpstreamDependency, ""
}

func mapHTTPStatus(status int) int {
	switch {
	case status == http.StatusUnauthorized:
		return http.StatusUnauthorized
	case status == http.StatusForbidden:
		return http.StatusForbidden
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return status
	default:
		return http.StatusFailedDependency
	}
}

func reasonForHTTPStatus(status int) hostcontract.ReasonCode {
	switch status {
	case http.StatusUnauthorized:
		return hostcontract.ReasonUnauthorized
	case http.StatusForbidden:
		return hostcontract.ReasonForbidden
	case http.StatusBadRequest:
		return hostcontract.ReasonInvalidArgument
	default:
		return hostcontract.ReasonUpstreamDependency
	}
}

func ensureInputKeys(input map[string]any, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range input {
		if _, ok := allowedSet[key]; !ok {
			return &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
	}
	return nil
}

func inputPositiveInt(input map[string]any, key string, fallback, maximum int) (int, error) {
	value, ok := input[key]
	if !ok {
		return fallback, nil
	}
	if valueNumber, isNumber := value.(json.Number); isNumber {
		parsed, err := valueNumber.Int64()
		if err != nil || parsed <= 0 || parsed > int64(maximum) {
			return 0, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
		return int(parsed), nil
	}
	number, ok := value.(float64)
	if !ok || math.Trunc(number) != number || number <= 0 || number > float64(maximum) {
		return 0, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return int(number), nil
}

func inputUUID(input map[string]any, key string) (string, error) {
	value, err := inputRequiredString(input, key)
	if err != nil {
		return "", err
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return parsed.String(), nil
}

func inputOptionalUUID(input map[string]any, key string) (string, error) {
	if _, ok := input[key]; !ok {
		return "", nil
	}
	return inputUUID(input, key)
}

func inputUUIDs(input map[string]any, key string) ([]string, error) {
	values, err := inputStrings(input, key)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
		canonical := parsed.String()
		if _, duplicate := seen[canonical]; duplicate {
			return nil, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
		seen[canonical] = struct{}{}
		result = append(result, canonical)
	}
	return result, nil
}

func inputStrings(input map[string]any, key string) ([]string, error) {
	raw, ok := input[key].([]any)
	if !ok || len(raw) == 0 {
		return nil, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	result := make([]string, 0, len(raw))
	for _, value := range raw {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
		}
		result = append(result, strings.TrimSpace(text))
	}
	return result, nil
}

func inputStringsOptional(input map[string]any, key string) ([]string, error) {
	if _, ok := input[key]; !ok {
		return nil, nil
	}
	return inputStrings(input, key)
}
func inputUUIDsOptional(input map[string]any, key string) ([]string, error) {
	if _, ok := input[key]; !ok {
		return nil, nil
	}
	return inputUUIDs(input, key)
}

func inputRequiredString(input map[string]any, key string) (string, error) {
	value, ok := input[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return strings.TrimSpace(value), nil
}

func inputOptionalString(input map[string]any, key string) (string, error) {
	value, exists := input[key]
	if !exists {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return strings.TrimSpace(text), nil
}

func inputOptionalObject(input map[string]any, key string) (map[string]any, error) {
	value, exists := input[key]
	if !exists {
		return nil, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return object, nil
}

func inputOptionalBool(input map[string]any, key string) (bool, error) {
	value, exists := input[key]
	if !exists {
		return false, nil
	}
	result, ok := value.(bool)
	if !ok {
		return false, &hostcontract.Error{Reason: hostcontract.ReasonInvalidArgument}
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
