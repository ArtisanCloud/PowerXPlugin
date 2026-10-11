package models

// backend/internal/entity/models/model.go

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement;comment:自增ID" json:"id"`
	TenantUuid string         `gorm:"type:uuid;not null;index;comment:租户UUID" json:"tenant_uuid"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;comment:软删除时间" json:"deleted_at,omitempty"`
}

type BaseNoTenantModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

const (
	TablePluginTenantExt                 = "plugin_tenant_ext"
	TableTemplate                        = "template"
	TablePluginSkills                    = "plugin_skills"
	TablePluginAgents                    = "plugin_agents"
	TablePluginCredentials               = "plugin_credentials"
	TablePrivacyDataClassifications      = "privacy_data_classifications"
	TablePrivacyConsentTokens            = "privacy_consent_tokens"
	TablePrivacyLifecycleEvents          = "privacy_lifecycle_events"
	TableSecurityBaselineChecklists      = "security_baseline_checklists"
	TableSecurityAuditReports            = "security_audit_reports"
	TableSecurityVulnerabilityAdvisory   = "security_vulnerability_advisories"
	TableSecurityAdvisoryDistributions   = "security_advisory_distributions"
	TableToolGrantRevocations            = "tool_grant_revocations"
	TableToolGrantUsageEvents            = "tool_grant_usage_events"
	TableIntegrationWebhookSubscriptions = "integration_webhook_subscriptions"
	TableIntegrationWebhookAttempts      = "integration_webhook_attempts"
	TableIntegrationSecrets              = "integration_secrets"
	TableIntegrationGrantMatrixOverrides = "integration_grant_matrix_overrides"
	TableMarketplaceListings             = "marketplace_listings"
	TableMarketplaceListingAssets        = "marketplace_listing_assets"
	TableMarketplaceListingVersions      = "marketplace_listing_versions"
	TableMarketplacePricingPlans         = "marketplace_pricing_plans"
	TableMarketplacePlanTiers            = "marketplace_plan_tiers"
	TableMarketplaceChecklistRuns        = "marketplace_checklist_runs"
	TableMarketplaceChecklistItems       = "marketplace_checklist_items"
	TableMarketplaceLicenses             = "marketplace_licenses"
	TableMarketplaceLicenseEvents        = "marketplace_license_events"
	TableMarketplaceTaxTransactions      = "marketplace_tax_transactions"
	TableCustomerAccounts                = "customer_accounts"
	TableCustomerContacts                = "customer_contacts"
	TableCustomerContactIdentities       = "customer_contact_identities"
	TableCustomerAuthIdentities          = "customer_auth_identities"
	TableCustomerTenantMemberships       = "customer_tenant_memberships"
	TableMiniAppEntries                  = "mini_app_entries"
	TableCustomerSessions                = "customer_sessions"
	TableCustomerLoginEvents             = "customer_login_events"
	TableMarketplaceUsageEnvelopes       = "marketplace_usage_envelopes"
	TableMarketplaceUsageAggregates      = "marketplace_usage_aggregates"
	TableMarketplaceRevenueReports       = "marketplace_revenue_share_reports"
	TableMarketplaceNotifications        = "marketplace_notifications"
	TableOperationsSupportChannels       = "operations_support_channels"
	TableOperationsSupportTickets        = "operations_support_tickets"
	TableOperationsIncidents             = "operations_incidents"
	TableOperationsIncidentUpdates       = "operations_incident_updates"
	TableOperationsIncidentChecklist     = "operations_incident_checklist"
	TableOperationsSupportTicketEvents   = "operations_support_ticket_events"
	TableOperationsReadinessItems        = "operations_readiness_checklist_items"
	TableOperationsSLAScores             = "operations_sla_profiles"
	TableOperationsSLAAdjustments        = "operations_sla_adjustments"
	TableAdminConsoleAuditEvents         = "admin_console_audit_events"
	TableAdminConsoleConfigChanges       = "admin_console_config_changes"
	TableAdminConsoleJobRuns             = "admin_console_job_runs"
	TableMetadataDictionaryNamespaces    = "metadata_dictionary_namespaces"
	TableMetadataDictionaryItems         = "metadata_dictionary_items"
	TableMetadataTaxonomies              = "metadata_taxonomies"
	TableMetadataTaxonomyNodes           = "metadata_taxonomy_nodes"
	TableMetadataTags                    = "metadata_tags"
	TableMetadataTagBindings             = "metadata_tag_bindings"
	TableMetadataResourceTypes           = "metadata_resource_types"
	TableIAMTenants                      = "iam_tenants"
	TableIAMUsers                        = "iam_users"
	TableIAMMembers                      = "iam_members"
	TableIAMRoles                        = "iam_roles"
	TableIAMPermissions                  = "iam_permissions"
	TableIAMDepartments                  = "iam_departments"
	TableIAMMemberRoles                  = "iam_member_roles"
	TableIAMRolePermissions              = "iam_role_permissions"
	TableIAMRefreshTokens                = "iam_refresh_tokens"
	TableIAMAuditLogs                    = "iam_audit_logs"
	TableIAMChannelSyncTasks             = "iam_channel_sync_tasks"
	TableLocalAISettings                 = "local_ai_settings"
	TablePluginSystemConfigs             = "plugin_system_configs"
	TableLocalKnowledgeSpaces            = "local_knowledge_spaces"
	TableLocalKnowledgeDocuments         = "local_knowledge_documents"
	TableLocalKnowledgeChunks            = "local_knowledge_chunks"
	TableLocalKnowledgeIngestionProfiles = "local_knowledge_ingestion_profile_versions"
	TableLocalKnowledgeIndexProfiles     = "local_knowledge_index_profile_versions"
	TableLocalKnowledgeRAGProfiles       = "local_knowledge_rag_profile_versions"
	TableLocalKnowledgeIngestionJobs     = "local_knowledge_ingestion_jobs"
	TableLocalKnowledgeJobChunks         = "local_knowledge_job_chunks"
	TableLocalKnowledgeIndexJobs         = "local_knowledge_index_jobs"
	TableLocalKnowledgeVectorIndexes     = "local_knowledge_vector_indexes"
	TableLocalKnowledgeArtifactBundles   = "local_knowledge_artifact_bundles"
	TableLocalKnowledgeAuditTrail        = "local_knowledge_audit_trail_entries"
	TableLocalKnowledgeCorpusChecks      = "local_knowledge_corpus_check_jobs"
	TableLocalKnowledgeDecayTasks        = "local_knowledge_decay_tasks"
	TableLocalKnowledgeDeltaJobs         = "local_knowledge_delta_jobs"
	TableLocalKnowledgeFeedbackCases     = "local_knowledge_feedback_cases"
	TableLocalKnowledgeFusionStrategies  = "local_knowledge_fusion_strategy_versions"
	TableLocalKnowledgeIAMSyncTasks      = "local_knowledge_iam_sync_tasks"
	TableLocalKnowledgePolicyTemplates   = "local_knowledge_policy_template_versions"
	TableLocalKnowledgeSourceConnectors  = "local_knowledge_source_connector_instances"
	TableLocalKnowledgeSourceCredentials = "local_knowledge_source_credentials"
	TableLocalKnowledgeSpaceSyncJobs     = "local_knowledge_space_sync_jobs"
	TableLocalKnowledgeReleasePolicies   = "local_knowledge_release_policies"
	TableLocalKnowledgeReleaseBatches    = "local_knowledge_release_batches"
	TableLocalKnowledgeRoutes            = "local_knowledge_routes"
	TableLocalKnowledgeKGNodes           = "local_knowledge_kg_nodes"
	TableLocalKnowledgeKGEdges           = "local_knowledge_kg_edges"
)
