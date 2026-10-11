// Package hostcontract defines the typed Host Contract Lab boundary shared by
// Framework clients and Skeleton transport. It deliberately contains no Core
// URL, credential, tenant, or local-store fallback.
package hostcontract

import (
	"errors"
	"strings"
	"time"
)

type Module string
type Operation string
type ReasonCode string

const (
	ModuleCache              Module = "cache"
	ModuleTaskCenter         Module = "taskcenter"
	ModuleIAM                Module = "iam"
	ModuleKnowledge          Module = "knowledge"
	ModuleMedia              Module = "media"
	ModuleAgent              Module = "agent"
	ModuleAI                 Module = "ai"
	ModuleCapabilityRegistry Module = "capability_registry"
	ModuleIntegrationGateway Module = "integration_gateway"
	ModuleSkills             Module = "skills"
	ModuleNotifications      Module = "notifications"
	ModulePluginRuntime      Module = "plugin_runtime"
)

const (
	OperationIAMTenant                        Operation = "tenant.get"
	OperationIAMMembersList                   Operation = "members.list"
	OperationIAMMemberGet                     Operation = "member.get"
	OperationIAMMembersBatchGet               Operation = "members.batch_get"
	OperationIAMMembersBatchResolve           Operation = "members.batch_resolve"
	OperationIAMMembersResolveDisplayNames    Operation = "members.resolve_display_names"
	OperationIAMDepartmentsList               Operation = "departments.list"
	OperationIAMRolesList                     Operation = "roles.list"
	OperationIAMPermissionsList               Operation = "permissions.list"
	OperationIAMAuthorizationCheck            Operation = "authorization.check"
	OperationKnowledgeSpacesList              Operation = "spaces.list"
	OperationKnowledgeSearch                  Operation = "search"
	OperationKnowledgeIndexJobGet             Operation = "index_job.get"
	OperationKnowledgeDocumentCreate          Operation = "document.create"
	OperationKnowledgeDocumentDelete          Operation = "document.delete"
	OperationKnowledgeIndexRebuild            Operation = "index.rebuild"
	OperationMediaAssetsList                  Operation = "assets.list"
	OperationAgentHealthSummary               Operation = "health.summary"
	OperationAgentFreeze                      Operation = "bridge.freeze"
	OperationAgentRecover                     Operation = "bridge.recover"
	OperationAgentRebalance                   Operation = "bridge.rebalance"
	OperationAIModelsList                     Operation = "models.list"
	OperationCapabilityRegistryList           Operation = "capabilities.list"
	OperationSkillInvoke                      Operation = "skill.invoke"
	OperationNotificationCreate               Operation = "notification.create"
	OperationIntegrationRoutesList            Operation = "routes.list"
	OperationIntegrationRouteInvoke           Operation = "route.invoke"
	OperationPluginRuntimeKnowledgeSpacesList Operation = "knowledge_spaces.list"
	OperationPluginRuntimeAgentsList          Operation = "agents.list"
	OperationPluginRuntimeAgentInstantiate    Operation = "agent.instantiate"
	OperationStatus                           Operation = "status"
)

const (
	ReasonInvalidArgument         ReasonCode = "HOST_CONTRACT_INVALID_ARGUMENT"
	ReasonUnsupportedModule       ReasonCode = "HOST_CONTRACT_UNSUPPORTED_MODULE"
	ReasonUnsupportedOperation    ReasonCode = "HOST_CONTRACT_UNSUPPORTED_OPERATION"
	ReasonTenantOverrideForbidden ReasonCode = "HOST_CONTRACT_TENANT_OVERRIDE_FORBIDDEN"
	ReasonUnauthorized            ReasonCode = "HOST_CONTRACT_UNAUTHORIZED"
	ReasonForbidden               ReasonCode = "HOST_CONTRACT_FORBIDDEN"
	ReasonUpstreamDependency      ReasonCode = "HOST_CONTRACT_UPSTREAM_DEPENDENCY"
	ReasonConfirmationRequired    ReasonCode = "HOST_CONTRACT_CONFIRMATION_REQUIRED"
)

// OperationDescriptor is the allowlisted semantic operation. CapabilityID is
// emitted into the audit/result envelope and is never supplied by the caller.
type OperationDescriptor struct {
	Module               Module
	Operation            Operation
	CapabilityID         string
	ReadOnly             bool
	RequiresConfirmation bool
}

// ProbeRequest deliberately has no tenant or capability field. The tenant is
// credential-derived, and the capability comes from the operation allowlist.
type ProbeRequest struct {
	Module    Module         `json:"module"`
	Operation Operation      `json:"operation"`
	Input     map[string]any `json:"input,omitempty"`
	Confirm   bool           `json:"confirm,omitempty"`
}

// ProbeResult is the stable transport DTO for Host Contract Lab responses.
// Result contains only module-specific typed DTO data after execution.
type ProbeResult struct {
	Module       Module         `json:"module"`
	Operation    Operation      `json:"operation"`
	ProviderMode string         `json:"provider_mode"`
	CapabilityID string         `json:"capability_id"`
	TraceID      string         `json:"trace_id,omitempty"`
	ReasonCode   ReasonCode     `json:"reason_code,omitempty"`
	Result       map[string]any `json:"result,omitempty"`
	ObservedAt   time.Time      `json:"observed_at"`
}

type AuditFields struct {
	Module       Module     `json:"module"`
	Operation    Operation  `json:"operation"`
	ProviderMode string     `json:"provider_mode"`
	CapabilityID string     `json:"capability_id"`
	TraceID      string     `json:"trace_id,omitempty"`
	ReasonCode   ReasonCode `json:"reason_code,omitempty"`
}

func (r ProbeResult) Audit() AuditFields {
	return AuditFields{Module: r.Module, Operation: r.Operation, ProviderMode: r.ProviderMode, CapabilityID: r.CapabilityID, TraceID: r.TraceID, ReasonCode: r.ReasonCode}
}

type ErrorEnvelope struct {
	ErrorCode  string     `json:"error_code"`
	ReasonCode ReasonCode `json:"reason_code"`
	TraceID    string     `json:"trace_id,omitempty"`
}

type Error struct {
	Reason ReasonCode
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Reason)
}

func CodeOf(err error) ReasonCode {
	var target *Error
	if errors.As(err, &target) && target != nil {
		return target.Reason
	}
	return ReasonUpstreamDependency
}

func LookupOperation(module Module, operation Operation) (OperationDescriptor, error) {
	module = Module(strings.TrimSpace(string(module)))
	operation = Operation(strings.TrimSpace(string(operation)))
	if _, ok := moduleOperations[module]; !ok {
		return OperationDescriptor{}, &Error{Reason: ReasonUnsupportedModule}
	}
	descriptor, ok := moduleOperations[module][operation]
	if !ok {
		return OperationDescriptor{}, &Error{Reason: ReasonUnsupportedOperation}
	}
	return descriptor, nil
}

func ValidateNoTenantOverride(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "tenant_uuid") {
				return &Error{Reason: ReasonTenantOverrideForbidden}
			}
			if err := ValidateNoTenantOverride(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range typed {
			if err := ValidateNoTenantOverride(item); err != nil {
				return err
			}
		}
	}
	return nil
}

var moduleOperations = map[Module]map[Operation]OperationDescriptor{
	ModuleCache: {
		OperationStatus: descriptor(ModuleCache, OperationStatus, "", true, false),
		"get":           descriptor(ModuleCache, "get", "com.corex.runtime.cache.read", true, false),
		"set":           descriptor(ModuleCache, "set", "com.corex.runtime.cache.manage", false, true),
		"delete":        descriptor(ModuleCache, "delete", "com.corex.runtime.cache.manage", false, true),
	},
	ModuleTaskCenter: {
		OperationStatus: descriptor(ModuleTaskCenter, OperationStatus, "", true, false),
		"create":        descriptor(ModuleTaskCenter, "create", "com.corex.runtime.taskcenter.manage", false, true),
		"get":           descriptor(ModuleTaskCenter, "get", "com.corex.runtime.taskcenter.read", true, false),
		"update":        descriptor(ModuleTaskCenter, "update", "com.corex.runtime.taskcenter.manage", false, true),
	},
	ModuleIAM: {
		OperationStatus:                        descriptor(ModuleIAM, OperationStatus, "", true, false),
		OperationIAMTenant:                     descriptor(ModuleIAM, OperationIAMTenant, "com.corex.iam.directory.read", true, false),
		OperationIAMMembersList:                descriptor(ModuleIAM, OperationIAMMembersList, "com.corex.iam.directory.read", true, false),
		OperationIAMMemberGet:                  descriptor(ModuleIAM, OperationIAMMemberGet, "com.corex.iam.members.read", true, false),
		OperationIAMMembersBatchGet:            descriptor(ModuleIAM, OperationIAMMembersBatchGet, "com.corex.iam.members.read", true, false),
		OperationIAMMembersBatchResolve:        descriptor(ModuleIAM, OperationIAMMembersBatchResolve, "com.corex.iam.directory.read", true, false),
		OperationIAMMembersResolveDisplayNames: descriptor(ModuleIAM, OperationIAMMembersResolveDisplayNames, "com.corex.iam.directory.read", true, false),
		OperationIAMDepartmentsList:            descriptor(ModuleIAM, OperationIAMDepartmentsList, "com.corex.iam.directory.read", true, false),
		OperationIAMRolesList:                  descriptor(ModuleIAM, OperationIAMRolesList, "com.corex.iam.directory.read", true, false),
		OperationIAMPermissionsList:            descriptor(ModuleIAM, OperationIAMPermissionsList, "com.corex.iam.directory.read", true, false),
		OperationIAMAuthorizationCheck:         descriptor(ModuleIAM, OperationIAMAuthorizationCheck, "com.corex.iam.authorization.check", true, false),
	},
	ModuleKnowledge: {
		OperationStatus:                  descriptor(ModuleKnowledge, OperationStatus, "", true, false),
		OperationKnowledgeSpacesList:     descriptor(ModuleKnowledge, OperationKnowledgeSpacesList, "com.corex.knowledge.directory.read", true, false),
		OperationKnowledgeSearch:         descriptor(ModuleKnowledge, OperationKnowledgeSearch, "com.corex.knowledge.search.read", true, false),
		OperationKnowledgeIndexJobGet:    descriptor(ModuleKnowledge, OperationKnowledgeIndexJobGet, "com.corex.knowledge.document.manage", true, false),
		OperationKnowledgeDocumentCreate: descriptor(ModuleKnowledge, OperationKnowledgeDocumentCreate, "com.corex.knowledge.document.manage", false, true),
		OperationKnowledgeDocumentDelete: descriptor(ModuleKnowledge, OperationKnowledgeDocumentDelete, "com.corex.knowledge.document.manage", false, true),
		OperationKnowledgeIndexRebuild:   descriptor(ModuleKnowledge, OperationKnowledgeIndexRebuild, "com.corex.knowledge.document.manage", false, true),
	},
	ModuleMedia: {
		"asset.get":                descriptor(ModuleMedia, "asset.get", "com.corex.media.assets.read", true, false),
		"asset.create":             descriptor(ModuleMedia, "asset.create", "com.corex.media.assets.manage", false, true),
		"asset.update":             descriptor(ModuleMedia, "asset.update", "com.corex.media.assets.manage", false, true),
		"asset.delete":             descriptor(ModuleMedia, "asset.delete", "com.corex.media.assets.manage", false, true),
		"asset.presign_upload":     descriptor(ModuleMedia, "asset.presign_upload", "com.corex.media.assets.manage", false, true),
		"asset.complete_upload":    descriptor(ModuleMedia, "asset.complete_upload", "com.corex.media.assets.manage", false, true),
		"asset.presign_download":   descriptor(ModuleMedia, "asset.presign_download", "com.corex.media.assets.read", true, false),
		"variant.create":           descriptor(ModuleMedia, "variant.create", "com.corex.media.assets.manage", false, true),
		"variant.get":              descriptor(ModuleMedia, "variant.get", "com.corex.media.assets.read", true, false),
		"variant.presign_upload":   descriptor(ModuleMedia, "variant.presign_upload", "com.corex.media.assets.manage", false, true),
		"variant.complete_upload":  descriptor(ModuleMedia, "variant.complete_upload", "com.corex.media.assets.manage", false, true),
		"variant.presign_download": descriptor(ModuleMedia, "variant.presign_download", "com.corex.media.assets.read", true, false),
		OperationStatus:            descriptor(ModuleMedia, OperationStatus, "com.corex.media.assets.read", true, false),
		OperationMediaAssetsList:   descriptor(ModuleMedia, OperationMediaAssetsList, "com.corex.media.assets.read", true, false),
	},
	ModuleAgent: {
		"session.create":            descriptor(ModuleAgent, "session.create", "com.corex.agent.session.manage", false, true),
		"sessions.list":             descriptor(ModuleAgent, "sessions.list", "com.corex.agent.session.manage", true, false),
		"session.get":               descriptor(ModuleAgent, "session.get", "com.corex.agent.session.manage", true, false),
		"session.rename":            descriptor(ModuleAgent, "session.rename", "com.corex.agent.session.manage", false, true),
		"session.archive":           descriptor(ModuleAgent, "session.archive", "com.corex.agent.session.manage", false, true),
		"session.delete":            descriptor(ModuleAgent, "session.delete", "com.corex.agent.session.manage", false, true),
		"session.message.append":    descriptor(ModuleAgent, "session.message.append", "com.corex.agent.session.manage", false, true),
		"session.messages.list":     descriptor(ModuleAgent, "session.messages.list", "com.corex.agent.session.manage", true, false),
		"session.invoke":            descriptor(ModuleAgent, "session.invoke", "com.corex.agent.invoke", false, true),
		"session.invocation.get":    descriptor(ModuleAgent, "session.invocation.get", "com.corex.agent.session.manage", true, false),
		"session.invocation.cancel": descriptor(ModuleAgent, "session.invocation.cancel", "com.corex.agent.invoke", false, true),
		OperationStatus:             descriptor(ModuleAgent, OperationStatus, "", true, false),
		OperationAgentHealthSummary: descriptor(ModuleAgent, OperationAgentHealthSummary, "com.corex.agent.lifecycle.manage", true, false),
		OperationAgentFreeze:        descriptor(ModuleAgent, OperationAgentFreeze, "com.corex.agent.lifecycle.manage", false, true),
		OperationAgentRecover:       descriptor(ModuleAgent, OperationAgentRecover, "com.corex.agent.lifecycle.manage", false, true),
		OperationAgentRebalance:     descriptor(ModuleAgent, OperationAgentRebalance, "com.corex.agent.lifecycle.manage", false, true),
	},
	ModuleAI: {
		"llm.invoke":          descriptor(ModuleAI, "llm.invoke", "", false, true),
		"embedding.invoke":    descriptor(ModuleAI, "embedding.invoke", "", false, true),
		OperationStatus:       descriptor(ModuleAI, OperationStatus, "", true, false),
		OperationAIModelsList: descriptor(ModuleAI, OperationAIModelsList, "", true, false),
	},
	ModuleCapabilityRegistry: {
		OperationStatus:                 descriptor(ModuleCapabilityRegistry, OperationStatus, "", true, false),
		OperationCapabilityRegistryList: descriptor(ModuleCapabilityRegistry, OperationCapabilityRegistryList, "", true, false),
	},
	ModuleIntegrationGateway: {
		OperationStatus:                 descriptor(ModuleIntegrationGateway, OperationStatus, "", true, false),
		OperationIntegrationRoutesList:  descriptor(ModuleIntegrationGateway, OperationIntegrationRoutesList, "", true, false),
		OperationIntegrationRouteInvoke: descriptor(ModuleIntegrationGateway, OperationIntegrationRouteInvoke, "", false, true),
	},
	ModuleSkills: {
		OperationStatus: descriptor(ModuleSkills, OperationStatus, "", true, false),
		// Skill-specific authorization is resolved by Core from skill_id; a
		// caller cannot nominate a broader capability here.
		OperationSkillInvoke: descriptor(ModuleSkills, OperationSkillInvoke, "", false, true),
	},
	ModuleNotifications: {
		OperationStatus:             descriptor(ModuleNotifications, OperationStatus, "", true, false),
		OperationNotificationCreate: descriptor(ModuleNotifications, OperationNotificationCreate, "com.corex.notifications.create", false, true),
	},
	ModulePluginRuntime: {
		OperationStatus: descriptor(ModulePluginRuntime, OperationStatus, "", true, false),
		OperationPluginRuntimeKnowledgeSpacesList: descriptor(ModulePluginRuntime, OperationPluginRuntimeKnowledgeSpacesList, "", true, false),
		OperationPluginRuntimeAgentsList:          descriptor(ModulePluginRuntime, OperationPluginRuntimeAgentsList, "", true, false),
		OperationPluginRuntimeAgentInstantiate:    descriptor(ModulePluginRuntime, OperationPluginRuntimeAgentInstantiate, "", false, true),
	},
}

func descriptor(module Module, operation Operation, capabilityID string, readOnly, requiresConfirmation bool) OperationDescriptor {
	return OperationDescriptor{Module: module, Operation: operation, CapabilityID: capabilityID, ReadOnly: readOnly, RequiresConfirmation: requiresConfirmation}
}
