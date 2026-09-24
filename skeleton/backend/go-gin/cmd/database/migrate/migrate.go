package migrate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	adminconsoleModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/admin_console"
	agentRegistryModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/agent_registry"
	customerModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	identitymodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	integrationModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	marketplaceModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/marketplace"
	metadataModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/metadata"
	operationsModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/operations"
	runtimeOpsModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/runtime_ops"
	securityModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/security"
	templateModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	toolgrantModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/tool_grant"
	localvector "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/knowledge/vectorstore"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"gorm.io/gorm"
)

var businessTables = []interface{}{
	&models.PluginCredential{},
	&models.PluginTenantExt{},
	&models.LocalAISetting{},
	&models.LocalKnowledgeSpace{},
	&models.LocalKnowledgeDocument{},
	&models.LocalKnowledgeRoute{},
	&models.LocalKnowledgeKGNode{},
	&models.LocalKnowledgeKGEdge{},
	&models.LocalKnowledgeChunk{},
	&models.LocalIngestionProfileVersion{},
	&models.LocalIndexProfileVersion{},
	&models.LocalRAGProfileVersion{},
	&models.LocalKnowledgeIngestionJob{},
	&models.LocalKnowledgeJobChunk{},
	&models.LocalKnowledgeIndexJob{},
	&models.LocalKnowledgeVectorIndex{},
	&models.LocalKnowledgeArtifactBundle{},
	&models.LocalKnowledgeAuditTrailEntry{},
	&models.LocalKnowledgeCorpusCheckJob{},
	&models.LocalKnowledgeDecayTask{},
	&models.LocalKnowledgeDeltaJob{},
	&models.LocalKnowledgeFeedbackCase{},
	&models.LocalKnowledgeFusionStrategyVersion{},
	&models.LocalKnowledgeIAMSyncTask{},
	&models.LocalKnowledgePolicyTemplateVersion{},
	&models.LocalKnowledgeSourceConnectorInstance{},
	&models.LocalKnowledgeSourceCredential{},
	&models.LocalKnowledgeSpaceSyncJob{},
	&models.LocalKnowledgeReleasePolicy{},
	&models.LocalKnowledgeReleaseBatch{},
	&agentRegistryModel.PluginSkill{},
	&agentRegistryModel.PluginAgent{},
	&templateModel.Template{},
	&marketplaceModel.Listing{},
	&marketplaceModel.ListingAsset{},
	&marketplaceModel.ListingVersion{},
	&marketplaceModel.ChecklistRun{},
	&marketplaceModel.ChecklistItem{},
	&marketplaceModel.PricingPlan{},
	&marketplaceModel.PlanTier{},
	&marketplaceModel.License{},
	&marketplaceModel.LicenseEvent{},
	&marketplaceModel.TaxTransaction{},
	&customerModel.CustomerAccount{},
	&customerModel.Contact{},
	&customerModel.ContactIdentity{},
	&customerModel.CustomerAuthIdentity{},
	&customerModel.CustomerTenantMembership{},
	&customerModel.MiniAppEntry{},
	&customerModel.CustomerSession{},
	&customerModel.CustomerLoginEvent{},
	&runtimeOpsModel.MCPSession{},
	&runtimeOpsModel.RuntimeAuditEvent{},
	&runtimeOpsModel.QuotaLedger{},
	&runtimeOpsModel.MarketplaceOverage{},
	&operationsModel.SupportChannel{},
	&operationsModel.SupportTicket{},
	&operationsModel.SupportTicketEvent{},
	&operationsModel.ReadinessChecklistItem{},
	&operationsModel.SLAProfile{},
	&operationsModel.SLAAdjustment{},
	&operationsModel.Incident{},
	&operationsModel.IncidentTimelineEntry{},
	&operationsModel.IncidentChecklistItem{},
	&integrationModel.SecretCredential{},
	&integrationModel.GrantMatrixOverride{},
	&securityModel.BaselineChecklist{},
	&securityModel.AuditReport{},
	&toolgrantModel.Revocation{},
	&toolgrantModel.UsageEvent{},
	&adminconsoleModel.AuditEvent{},
	&adminconsoleModel.ConfigChange{},
	&adminconsoleModel.JobRun{},
	&metadataModel.DictionaryNamespace{},
	&metadataModel.DictionaryItem{},
	&metadataModel.Taxonomy{},
	&metadataModel.TaxonomyNode{},
	&metadataModel.Tag{},
	&metadataModel.TagBinding{},
	&metadataModel.ResourceType{},
}

var iamTables = []interface{}{
	&identitymodel.Tenant{},
	&identitymodel.User{},
	&identitymodel.Member{},
	&identitymodel.Role{},
	&identitymodel.Permission{},
	&identitymodel.Department{},
	&identitymodel.MemberRole{},
	&identitymodel.RolePermission{},
	&identitymodel.RefreshToken{},
	&identitymodel.AuditLog{},
	&identitymodel.FederatedExternalIdentity{},
	&identitymodel.FederatedBinding{},
	&identitymodel.FederatedLoginChallenge{},
	&identitymodel.FederatedRiskEvent{},
	&identitymodel.ChannelSyncTask{},
}

// MigratePluginModels 只做 AutoMigrate（最小实现）
func MigratePluginModels(ctx context.Context, db *gorm.DB, includeIAM bool) error {
	return MigratePluginModelsWithConfig(ctx, db, nil, includeIAM)
}

// MigratePluginModelsWithConfig keeps the normal model migration compatible
// with callers that do not have runtime configuration, while make migrate can
// additionally provision the explicit local pgvector infrastructure.
func MigratePluginModelsWithConfig(ctx context.Context, db *gorm.DB, cfg *config.Config, includeIAM bool) error {
	if db == nil {
		return nil
	}
	tables := append([]interface{}{}, businessTables...)
	if isSQLite(db) {
		tables = filterSQLiteIncompatibleTables(tables)
	}
	if includeIAM {
		tables = append(tables, iamTables...)
	}
	if len(tables) == 0 {
		return nil
	}
	// Existing databases can have an earlier nullable UUID column. Backfill it
	// before AutoMigrate attempts to enforce the model's NOT NULL constraint.
	// New tables are intentionally skipped here and created by AutoMigrate.
	if err := ensureTemplateUUIDs(ctx, db); err != nil {
		return err
	}
	if err := ensureMetadataTagBindingUUIDs(ctx, db); err != nil {
		return err
	}
	if err := ensureContactIdentityChannelDictionaryItemUUID(ctx, db); err != nil {
		return err
	}
	if includeIAM {
		if err := ensureIAMIdentityUUIDs(ctx, db); err != nil {
			return err
		}
	}
	// Normalize the former `name` column before AutoMigrate sees the current
	// `space_name` model.  The old field is not a compatibility field: it is
	// copied once and removed in the same migration run.
	if err := migrateLocalKnowledgeSpaceName(ctx, db); err != nil {
		return err
	}
	if err := ensureLocalKnowledgeVectorIndexTenant(ctx, db); err != nil {
		return err
	}
	if err := safeAutoMigrate(ctx, db, tables); err != nil {
		return err
	}
	if err := ensureLocalKnowledgePGVector(ctx, db, cfg); err != nil {
		return err
	}
	if err := removeObsoleteLocalKnowledgeSchema(ctx, db); err != nil {
		return err
	}
	if err := migrateLocalKnowledgeProfileReferences(ctx, db); err != nil {
		return err
	}
	if err := migrateLocalAISettingSources(ctx, db); err != nil {
		return err
	}
	if err := removeObsoleteAICatalogSource(ctx, db); err != nil {
		return err
	}
	if includeIAM {
		if err := ensureIAMConstraints(ctx, db); err != nil {
			return err
		}
		if err := backfillIAMUUIDRelations(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

func ensureLocalKnowledgePGVector(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	if cfg == nil || !cfg.LocalPGVectorEnabled() {
		return nil
	}
	pg := cfg.Knowledge.VectorStore.PGVector
	log.Printf("[migrate] local knowledge pgvector target=%s.%s dimensions=%d", pg.Schema, pg.Table, pg.Dimensions)
	if err := localvector.EnsurePGVectorTable(ctx, db, localvector.PGVectorConfig{
		Schema:     pg.Schema,
		Table:      pg.Table,
		Dimensions: pg.Dimensions,
		Lists:      pg.Lists,
	}, cfg.Database.Schema); err != nil {
		return fmt.Errorf("local knowledge pgvector migration failed: %w", err)
	}
	log.Printf("[migrate] local knowledge pgvector ready: %s.%s", pg.Schema, pg.Table)
	return nil
}

// ensureLocalKnowledgeVectorIndexTenant upgrades the short-lived pre-Dense
// index catalog before AutoMigrate applies the non-null tenant UUID contract.
// The backfill is derived only from the owning local knowledge space; an
// orphan index is rejected rather than assigned a synthetic tenant.
func ensureLocalKnowledgeVectorIndexTenant(_ context.Context, db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&models.LocalKnowledgeVectorIndex{}) {
		return nil
	}
	indexTable := models.LocalKnowledgeVectorIndex{}.TableName()
	if !db.Migrator().HasColumn(&models.LocalKnowledgeVectorIndex{}, "TenantUUID") {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_uuid varchar(36)", indexTable)).Error; err != nil {
			return err
		}
	}
	spaceTable := models.LocalKnowledgeSpace{}.TableName()
	if err := db.Exec(fmt.Sprintf("UPDATE %s SET tenant_uuid = (SELECT tenant_uuid FROM %s WHERE %s.uuid = %s.space_uuid) WHERE tenant_uuid IS NULL OR CAST(tenant_uuid AS TEXT) = ''", indexTable, spaceTable, spaceTable, indexTable)).Error; err != nil {
		return err
	}
	var orphanCount int64
	if err := db.Table(indexTable).Where("tenant_uuid IS NULL OR CAST(tenant_uuid AS TEXT) = ''").Count(&orphanCount).Error; err != nil {
		return err
	}
	if orphanCount != 0 {
		return fmt.Errorf("local knowledge vector indexes have %d tenantless rows", orphanCount)
	}
	return nil
}

// removeObsoleteLocalKnowledgeSchema removes the abandoned first-generation
// local knowledge projection. The formal Core-aligned version and job tables
// are the only supported storage shape.
func removeObsoleteLocalKnowledgeSchema(_ context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	for _, table := range []string{"local_knowledge_profiles", "local_knowledge_jobs", "local_knowledge_chunk_vectors"} {
		if db.Migrator().HasTable(table) {
			if err := db.Migrator().DropTable(table); err != nil {
				return err
			}
		}
	}
	return nil
}

// migrateLocalKnowledgeSpaceName removes the abandoned `name` storage field.
// Existing values are copied into the Core-aligned `space_name` field before
// the old column is dropped.  A blank historical name is rejected explicitly
// instead of allowing an invalid space to survive the migration.
func migrateLocalKnowledgeSpaceName(_ context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	table := models.LocalKnowledgeSpace{}.TableName()
	// Inspect physical columns rather than asking GORM to resolve a current Go
	// field.  `name` deliberately no longer exists in LocalKnowledgeSpace.
	hasName, err := hasPhysicalColumn(db, table, "name")
	if err != nil {
		return err
	}
	if !hasName {
		return nil
	}
	hasSpaceName, err := hasPhysicalColumn(db, table, "space_name")
	if err != nil {
		return err
	}
	if !hasSpaceName {
		// It must initially be nullable: SQLite cannot add a NOT NULL column to
		// a populated table, and PostgreSQL must first receive the historical
		// values.  PostgreSQL gets NOT NULL restored below; SQLite's subsequent
		// AutoMigrate rebuild applies the current model constraint.
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN space_name varchar(128)", table)).Error; err != nil {
			return err
		}
	}
	if err := db.Exec(fmt.Sprintf("UPDATE %s SET space_name = name WHERE space_name IS NULL OR TRIM(space_name) = ''", table)).Error; err != nil {
		return err
	}
	var missing int64
	if err := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE space_name IS NULL OR TRIM(space_name) = ''", table)).Scan(&missing).Error; err != nil {
		return err
	}
	if missing != 0 {
		return fmt.Errorf("local knowledge space migration found %d rows without a usable legacy name", missing)
	}
	if !isSQLite(db) {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN space_name SET NOT NULL", table)).Error; err != nil {
			return err
		}
	}
	// Use native DDL rather than GORM's SQLite table-recreation path: the
	// latter resolves `name` against the current Go model and panics because
	// that legacy field is deliberately no longer present there.
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN name", table)).Error; err != nil {
		return err
	}
	return nil
}

func hasPhysicalColumn(db *gorm.DB, table, column string) (bool, error) {
	baseTable := strings.Trim(strings.TrimSpace(table[strings.LastIndex(table, ".")+1:]), `"`)
	if db != nil && db.Dialector != nil {
		switch db.Dialector.Name() {
		case "postgres":
			var count int64
			schema := models.Schema()
			if err := db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?`, schema, baseTable, column).Scan(&count).Error; err != nil {
				return false, err
			}
			return count != 0, nil
		case "sqlite":
			var count int64
			if err := db.Raw(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, baseTable, column).Scan(&count).Error; err != nil {
				return false, err
			}
			return count != 0, nil
		}
	}
	columns, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		// The SQLite test migration intentionally skips local-knowledge tables.
		// A missing table therefore means there is no legacy column to migrate.
		if isSQLite(db) && strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return false, nil
		}
		return false, err
	}
	for _, item := range columns {
		if strings.EqualFold(item.Name(), column) {
			return true, nil
		}
	}
	return false, nil
}

// migrateLocalKnowledgeProfileReferences promotes old UUID links to the same
// logical profile-key contract used by PowerX Core.  It runs before the UUID
// columns are dropped, so existing local spaces retain their selected policy.
func migrateLocalKnowledgeProfileReferences(_ context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	table := models.LocalKnowledgeSpace{}.TableName()
	columns := []struct{ oldColumn, newColumn, profileTable string }{
		{"ingestion_profile_uuid", "ingestion_profile_key", models.LocalIngestionProfileVersion{}.TableName()},
		{"index_profile_uuid", "index_profile_key", models.LocalIndexProfileVersion{}.TableName()},
		{"rag_profile_uuid", "rag_profile_key", models.LocalRAGProfileVersion{}.TableName()},
	}
	for _, item := range columns {
		hasOld, err := hasPhysicalColumn(db, table, item.oldColumn)
		if err != nil {
			return err
		}
		hasNew, err := hasPhysicalColumn(db, table, item.newColumn)
		if err != nil {
			return err
		}
		if !hasOld || !hasNew {
			continue
		}
		query := fmt.Sprintf("UPDATE %s AS space SET %s = profile.profile_key FROM %s AS profile WHERE space.%s = profile.uuid AND (space.%s = '' OR space.%s = 'default')", table, item.newColumn, item.profileTable, item.oldColumn, item.newColumn, item.newColumn)
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}
	for _, column := range []string{"profile_uuid", "ingestion_profile_uuid", "index_profile_uuid", "rag_profile_uuid"} {
		hasColumn, err := hasPhysicalColumn(db, table, column)
		if err != nil {
			return err
		}
		if hasColumn {
			if err := dropPhysicalColumn(db, table, column); err != nil {
				return err
			}
		}
	}
	hasDescription, err := hasPhysicalColumn(db, table, "description")
	if err != nil {
		return err
	}
	if hasDescription {
		if err := dropPhysicalColumn(db, table, "description"); err != nil {
			return err
		}
	}
	return nil
}

func dropPhysicalColumn(db *gorm.DB, table, column string) error {
	return db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column)).Error
}

// migrateLocalAISettingSources splits the old single profile slot into local
// and PowerX source partitions. Existing rows retain their previous plugin
// configuration and are therefore assigned to the local partition.
func migrateLocalAISettingSources(ctx context.Context, db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.LocalAISetting{}, "source") {
		return nil
	}
	table := models.LocalAISetting{}.TableName()
	if err := db.WithContext(ctx).Table(table).Where("source IS NULL OR source = ''").Update("source", "local").Error; err != nil {
		return err
	}
	const legacyIndex = "uk_local_ai_settings_tenant_env_modality"
	const sourceIndex = "uk_local_ai_settings_tenant_env_modality_source"
	if db.Migrator().HasIndex(&models.LocalAISetting{}, legacyIndex) {
		if err := db.Migrator().DropIndex(&models.LocalAISetting{}, legacyIndex); err != nil {
			return err
		}
	}
	if !db.Migrator().HasIndex(&models.LocalAISetting{}, sourceIndex) {
		if err := db.Migrator().CreateIndex(&models.LocalAISetting{}, sourceIndex); err != nil {
			return err
		}
	}
	return nil
}

// removeObsoleteAICatalogSource removes the previously persisted page-only
// catalog preference. It is idempotent: the generic system-config table is
// kept for future settings, while this no-longer-valid key is removed.
func removeObsoleteAICatalogSource(ctx context.Context, db *gorm.DB) error {
	if db.Migrator().HasTable(&models.PluginSystemConfig{}) {
		if err := db.WithContext(ctx).Where("key = ?", "ai.catalog_source").Delete(&models.PluginSystemConfig{}).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasColumn(&models.LocalAISetting{}, "catalog_source") {
		return db.Migrator().DropColumn(&models.LocalAISetting{}, "catalog_source")
	}
	return nil
}

// backfillIAMUUIDRelations upgrades existing local IAM rows without accepting
// numeric IDs at the API boundary. It is idempotent and only fills an empty
// UUID relation from the pre-existing private numeric key.
func backfillIAMUUIDRelations(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	// AutoMigrate adds UUID columns but does not run BeforeCreate for rows that
	// predate the columns. Populate the business objects first, then derive all
	// UUID relations from those stable values.
	if err := backfillIAMObjectUUIDs(ctx, db); err != nil {
		return err
	}
	var departments []identitymodel.Department
	if err := db.WithContext(ctx).Find(&departments).Error; err != nil {
		return err
	}
	departmentUUIDs := make(map[uint64]string, len(departments))
	for _, department := range departments {
		departmentUUIDs[department.ID] = strings.TrimSpace(department.UUID)
	}
	for _, department := range departments {
		if department.ParentID == nil || strings.TrimSpace(derefString(department.ParentDepartmentUUID)) != "" {
			continue
		}
		if parentUUID := strings.TrimSpace(departmentUUIDs[*department.ParentID]); parentUUID != "" {
			if err := db.WithContext(ctx).Model(&identitymodel.Department{}).Where("id = ?", department.ID).Update("parent_department_uuid", parentUUID).Error; err != nil {
				return err
			}
		}
	}

	var members []identitymodel.Member
	if err := db.WithContext(ctx).Find(&members).Error; err != nil {
		return err
	}
	memberUUIDs := make(map[uint64]string, len(members))
	for _, member := range members {
		memberUUIDs[member.ID] = strings.TrimSpace(member.UUID)
		if member.DepartmentID != nil && strings.TrimSpace(derefString(member.DepartmentUUID)) == "" {
			if departmentUUID := strings.TrimSpace(departmentUUIDs[*member.DepartmentID]); departmentUUID != "" {
				if err := db.WithContext(ctx).Model(&identitymodel.Member{}).Where("id = ?", member.ID).Update("department_uuid", departmentUUID).Error; err != nil {
					return err
				}
			}
		}
	}

	var roles []identitymodel.Role
	if err := db.WithContext(ctx).Find(&roles).Error; err != nil {
		return err
	}
	roleUUIDs := make(map[uint64]string, len(roles))
	for _, role := range roles {
		roleUUIDs[role.ID] = strings.TrimSpace(role.UUID)
	}
	var permissions []identitymodel.Permission
	if err := db.WithContext(ctx).Find(&permissions).Error; err != nil {
		return err
	}
	permissionUUIDs := make(map[uint64]string, len(permissions))
	for _, permission := range permissions {
		permissionUUIDs[permission.ID] = strings.TrimSpace(permission.UUID)
	}
	var memberRoles []identitymodel.MemberRole
	if err := db.WithContext(ctx).Find(&memberRoles).Error; err != nil {
		return err
	}
	for _, relation := range memberRoles {
		updates := map[string]any{}
		if strings.TrimSpace(relation.MemberUUID) == "" && strings.TrimSpace(memberUUIDs[relation.UserID]) != "" {
			updates["member_uuid"] = memberUUIDs[relation.UserID]
		}
		if strings.TrimSpace(relation.RoleUUID) == "" && strings.TrimSpace(roleUUIDs[relation.RoleID]) != "" {
			updates["role_uuid"] = roleUUIDs[relation.RoleID]
		}
		if len(updates) > 0 {
			if err := db.WithContext(ctx).Model(&identitymodel.MemberRole{}).Where("member_id = ? AND role_id = ?", relation.UserID, relation.RoleID).Updates(updates).Error; err != nil {
				return err
			}
		}
	}
	var rolePermissions []identitymodel.RolePermission
	if err := db.WithContext(ctx).Find(&rolePermissions).Error; err != nil {
		return err
	}
	for _, relation := range rolePermissions {
		updates := map[string]any{}
		if strings.TrimSpace(relation.RoleUUID) == "" && strings.TrimSpace(roleUUIDs[relation.RoleID]) != "" {
			updates["role_uuid"] = roleUUIDs[relation.RoleID]
		}
		if strings.TrimSpace(relation.PermissionUUID) == "" && strings.TrimSpace(permissionUUIDs[relation.PermissionID]) != "" {
			updates["permission_uuid"] = permissionUUIDs[relation.PermissionID]
		}
		if len(updates) > 0 {
			if err := db.WithContext(ctx).Model(&identitymodel.RolePermission{}).Where("role_id = ? AND permission_id = ?", relation.RoleID, relation.PermissionID).Updates(updates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func backfillIAMObjectUUIDs(ctx context.Context, db *gorm.DB) error {
	tables := []string{
		identitymodel.Tenant{}.TableName(),
		identitymodel.User{}.TableName(),
		identitymodel.Member{}.TableName(),
		identitymodel.Department{}.TableName(),
		identitymodel.Role{}.TableName(),
		identitymodel.Permission{}.TableName(),
	}
	for _, table := range tables {
		var rows []struct {
			ID   uint64
			UUID string
		}
		if err := db.WithContext(ctx).Table(table).Select("id, uuid").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if validIAMUUID(row.UUID) {
				continue
			}
			if err := db.WithContext(ctx).Table(table).Where("id = ?", row.ID).Update("uuid", uuid.NewString()).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func validIAMUUID(value string) bool {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	return err == nil && parsed.String() == strings.ToLower(strings.TrimSpace(value))
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func safeAutoMigrate(ctx context.Context, db *gorm.DB, tables []interface{}) error {
	for _, tbl := range tables {
		if err := migrateWithTolerance(ctx, db, tbl); err != nil {
			return err
		}
	}
	return nil
}

func migrateWithTolerance(ctx context.Context, db *gorm.DB, table interface{}) error {
	const maxRetries = 5
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := db.WithContext(ctx).AutoMigrate(table); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == duplicateObjectCode {
				log.Printf("[migrate] duplicate constraint for %T, skipping: %s", table, pgErr.ConstraintName)
				return nil
			}
			// 部分包装错误解析不到 pgErr，但文本包含 already exists/constraint，直接跳过以保证幂等
			if strings.Contains(err.Error(), "already exists") && strings.Contains(err.Error(), "constraint") {
				log.Printf("[migrate] duplicate constraint (fallback) for %T, skipping: %v", table, err)
				return nil
			}
			handled, handleErr := tryHandleAutoMigrateError(ctx, db, table, err)
			if !handled {
				// 兜底：即便未被处理但仍是 42710，也不让迁移失败
				if errors.As(err, &pgErr) && pgErr.Code == duplicateObjectCode {
					log.Printf("[migrate] duplicate constraint (post-handle) for %T, skipping: %v", table, err)
					return nil
				}
				return fmt.Errorf("auto migrate %T failed: %w", table, err)
			}
			if handleErr != nil {
				return handleErr
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("auto migrate retries exceeded for %T", table)
}

func tryHandleAutoMigrateError(ctx context.Context, db *gorm.DB, table interface{}, migrateErr error) (bool, error) {
	if !strings.EqualFold(db.Dialector.Name(), "postgres") {
		return false, nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(migrateErr, &pgErr) {
		return false, nil
	}
	if pgErr.Code != duplicateObjectCode {
		return false, nil
	}
	if strings.TrimSpace(pgErr.ConstraintName) == "" {
		log.Printf("[migrate] duplicate object reported without constraint name, cannot auto fix: %v", migrateErr)
		return false, nil
	}
	tableName, err := resolveTableName(db, table)
	if err != nil {
		return true, err
	}
	if err := dropConstraintIfExists(ctx, db, tableName, pgErr.ConstraintName); err != nil {
		return true, err
	}
	log.Printf("[migrate] dropped duplicate constraint %q on %s, retrying migrate", pgErr.ConstraintName, tableName)
	if retryErr := db.WithContext(ctx).AutoMigrate(table); retryErr != nil {
		return true, retryErr
	}
	return true, nil
}

const duplicateObjectCode = "42710"

func dropConstraintIfExists(ctx context.Context, db *gorm.DB, tableName, constraintName string) error {
	clean := sanitizeConstraintName(constraintName)
	query := fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, tableName, quoteIdentifier(clean))
	return db.WithContext(ctx).Exec(query).Error
}

func resolveTableName(db *gorm.DB, table interface{}) (string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(table); err != nil {
		return "", err
	}
	if stmt.Schema == nil || stmt.Schema.Table == "" {
		return "", fmt.Errorf("failed to resolve table name for %T", table)
	}
	return stmt.Schema.Table, nil
}

func quoteIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, "\"", "\"\"")
	return fmt.Sprintf(`"%s"`, escaped)
}

// sanitizeConstraintName strips surrounding quotes and schema prefixes that may appear in pg error messages.
func sanitizeConstraintName(name string) string {
	trimmed := strings.Trim(name, `"`)
	// drop all quotes that may be embedded (fk_"public"_foo)
	trimmed = strings.ReplaceAll(trimmed, `"`, "")
	// remove schema prefix like public_ or public__
	trimmed = strings.TrimPrefix(trimmed, `public_`)
	trimmed = strings.ReplaceAll(trimmed, `public__`, ``)
	return trimmed
}

func isSQLite(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return false
	}
	return strings.EqualFold(db.Dialector.Name(), "sqlite")
}

func filterSQLiteIncompatibleTables(tables []interface{}) []interface{} {
	filtered := make([]interface{}, 0, len(tables))
	skipped := 0
	for _, tbl := range tables {
		if !isSQLiteSafeTable(tbl) {
			skipped++
			continue
		}
		filtered = append(filtered, tbl)
	}
	if skipped > 0 {
		log.Printf("[migrate] sqlite 环境仅迁移 IAM + 插件核心表，跳过 %d 张业务表", skipped)
	}
	return filtered
}

func isSQLiteSafeTable(tbl interface{}) bool {
	switch tbl.(type) {
	case *models.PluginCredential,
		*models.PluginTenantExt,
		*templateModel.Template,
		*runtimeOpsModel.MCPSession:
		return true
	default:
		return false
	}
}

func ensureIAMConstraints(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := ensureIAMIdentityUUIDs(ctx, db); err != nil {
		return err
	}
	departments := models.S(models.TableIAMDepartments)
	users := models.S(models.TableIAMMembers)
	audits := models.S(models.TableIAMAuditLogs)
	roles := models.S(models.TableIAMRoles)
	rolePerms := models.S(models.TableIAMRolePermissions)
	memberRoles := models.S(models.TableIAMMemberRoles)

	isSqlite := isSQLite(db)
	// SQLite 对表达式索引（lower(col)）在部分环境/驱动下会报 `near "(": syntax error`。
	// 这里用 COLLATE NOCASE 兼容本地开发，达到大小写不敏感的唯一性约束效果。
	deptCodeExpr := "lower(code)"
	memberUsernameExpr := "lower(username)"
	roleCodeExpr := "lower(code)"
	if isSqlite {
		deptCodeExpr = "code COLLATE NOCASE"
		memberUsernameExpr = "username COLLATE NOCASE"
		roleCodeExpr = "code COLLATE NOCASE"
	}

	statements := []string{
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_departments_tenant_code ON %s (tenant_uuid, %s)`, departments, deptCodeExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_departments_tenant_parent ON %s (tenant_uuid, parent_id)`, departments),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_users_tenant_account ON %s (tenant_uuid, user_id)`, users),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_users_tenant_username ON %s (tenant_uuid, %s)`, users, memberUsernameExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_users_tenant_status ON %s (tenant_uuid, status)`, users),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_tenant_created ON %s (tenant_uuid, created_at DESC)`, audits),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_actor_created ON %s (actor_member_id, created_at DESC)`, audits),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_roles_tenant_code ON %s (tenant_uuid, %s)`, roles, roleCodeExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_roles_scope ON %s (tenant_uuid, scope_type)`, roles),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_role_permissions_version ON %s (role_id, policy_version)`, rolePerms),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_role_permissions_tenant_uuid ON %s (tenant_uuid)`, rolePerms),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_member_roles_role ON %s (role_id)`, memberRoles),
	}

	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if err := execIgnoreExists(ctx, db, stmt); err != nil {
			return err
		}
	}
	if err := ensureChannelSyncTaskPayloadJSONB(ctx, db); err != nil {
		return err
	}
	return backfillRolePermissionTenant(ctx, db)
}

func ensureIAMIdentityUUIDs(ctx context.Context, db *gorm.DB) error {
	userTable := models.S(models.TableIAMUsers)
	memberTable := models.S(models.TableIAMMembers)
	roleTable := models.S(models.TableIAMRoles)
	departmentTable := models.S(models.TableIAMDepartments)
	permissionTable := models.S(models.TableIAMPermissions)
	entries := []struct {
		table string
		seed  string
		index string
	}{
		{userTable, "iam_users", "idx_iam_users_uuid"},
		{memberTable, "iam_members", "idx_iam_members_uuid"},
		{roleTable, "iam_roles", "idx_iam_roles_uuid"},
		{departmentTable, "iam_departments", "idx_iam_departments_uuid"},
		{permissionTable, "iam_permissions", "idx_iam_permissions_uuid"},
	}
	for _, entry := range entries {
		// Legacy installations can have a subset of IAM tables. Each existing
		// table must be repaired independently; one missing table must not skip
		// UUID backfill for roles, departments, or permissions.
		exists, err := iamTableExists(ctx, db, entry.table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := ensureUUIDColumn(ctx, db, entry.table); err != nil {
			return err
		}
		if err := backfillIdentityUUID(ctx, db, entry.table, entry.seed); err != nil {
			return err
		}
		stmt := fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (uuid) WHERE uuid IS NOT NULL`, entry.index, entry.table)
		if err := execIgnoreExists(ctx, db, stmt); err != nil {
			return err
		}
	}
	return nil
}

func iamTableExists(ctx context.Context, db *gorm.DB, tableName string) (bool, error) {
	if isSQLite(db) {
		return db.Migrator().HasTable(tableName), nil
	}
	var relation *string
	if err := db.WithContext(ctx).Raw("SELECT to_regclass(?)", tableName).Scan(&relation).Error; err != nil {
		return false, err
	}
	return relation != nil && strings.TrimSpace(*relation) != "", nil
}

func ensureUUIDColumn(ctx context.Context, db *gorm.DB, tableName string) error {
	// SQLite HasColumn may match an unquoted tenant_uuid column for uuid.
	// Inspect exact names rather than a DDL substring match.
	columns, err := db.WithContext(ctx).Migrator().ColumnTypes(tableName)
	if err != nil {
		return err
	}
	for _, column := range columns {
		if column.Name() == "uuid" {
			return nil
		}
	}
	columnType := "uuid"
	if isSQLite(db) {
		columnType = "TEXT"
	}
	stmt := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN uuid %s`, tableName, columnType)
	if strings.EqualFold(db.Dialector.Name(), "postgres") {
		stmt = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS uuid %s`, tableName, columnType)
	}
	if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "duplicate column") || strings.Contains(lower, "already exists") {
			return nil
		}
		return fmt.Errorf("ensure uuid column on %s failed: %w", tableName, err)
	}
	return nil
}

func backfillIdentityUUID(ctx context.Context, db *gorm.DB, tableName, seed string) error {
	if isSQLite(db) {
		stmt := fmt.Sprintf(`
UPDATE %s
SET uuid =
  substr(lower(hex(randomblob(16))), 1, 8) || '-' ||
  substr(lower(hex(randomblob(16))), 1, 4) || '-' ||
  substr(lower(hex(randomblob(16))), 1, 4) || '-' ||
  substr(lower(hex(randomblob(16))), 1, 4) || '-' ||
  substr(lower(hex(randomblob(16))), 1, 12)
WHERE uuid IS NULL OR trim(uuid) = ''
`, tableName)
		if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return fmt.Errorf("backfill uuid on %s failed: %w", tableName, err)
		}
		return nil
	}
	stmt := fmt.Sprintf(`
UPDATE %s
SET uuid = (
  substr(md5('%s:' || id::text), 1, 8) || '-' ||
  substr(md5('%s:' || id::text), 9, 4) || '-' ||
  substr(md5('%s:' || id::text), 13, 4) || '-' ||
  substr(md5('%s:' || id::text), 17, 4) || '-' ||
  substr(md5('%s:' || id::text), 21, 12)
)::uuid
WHERE uuid IS NULL OR btrim(uuid::text) = ''
`, tableName, seed, seed, seed, seed, seed)
	if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
		return fmt.Errorf("backfill uuid on %s failed: %w", tableName, err)
	}
	return nil
}

func ensureChannelSyncTaskPayloadJSONB(ctx context.Context, db *gorm.DB) error {
	if db == nil || isSQLite(db) {
		return nil
	}
	tableName := models.S(models.TableIAMChannelSyncTasks)
	columns := []string{"request_payload", "result_payload"}
	for _, column := range columns {
		colType, err := queryColumnDataType(ctx, db, tableName, column)
		if err != nil {
			return err
		}
		if strings.EqualFold(colType, "jsonb") {
			continue
		}
		stmt := fmt.Sprintf(
			`ALTER TABLE %s ALTER COLUMN %s TYPE jsonb USING CASE WHEN %s IS NULL OR btrim(%s::text) = '' THEN '{}'::jsonb ELSE %s::jsonb END`,
			tableName, quoteIdentifier(column), quoteIdentifier(column), quoteIdentifier(column), quoteIdentifier(column),
		)
		if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
				log.Printf("[migrate] column missing, skip payload jsonb migration: %v", err)
				continue
			}
			return fmt.Errorf("ensure channel sync payload jsonb failed: %w", err)
		}
	}
	return nil
}

func queryColumnDataType(ctx context.Context, db *gorm.DB, tableName, columnName string) (string, error) {
	parts := strings.Split(strings.TrimSpace(tableName), ".")
	schema := "public"
	table := strings.TrimSpace(tableName)
	if len(parts) == 2 {
		schema = strings.Trim(parts[0], `"`)
		table = strings.Trim(parts[1], `"`)
	} else {
		table = strings.Trim(tableName, `"`)
	}
	var dataType string
	err := db.WithContext(ctx).
		Raw(`
SELECT data_type
FROM information_schema.columns
WHERE table_schema = ? AND table_name = ? AND column_name = ?
LIMIT 1
`, schema, table, strings.TrimSpace(columnName)).
		Scan(&dataType).Error
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dataType) == "" {
		return "", fmt.Errorf("column not found: %s.%s", tableName, columnName)
	}
	return strings.TrimSpace(dataType), nil
}

func execIgnoreExists(ctx context.Context, db *gorm.DB, stmt string) error {
	if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			log.Printf("[migrate] index exists, skip: %s", stmt)
			return nil
		}
		return fmt.Errorf("exec stmt failed: %s: %w", stmt, err)
	}
	return nil
}

func backfillRolePermissionTenant(ctx context.Context, db *gorm.DB) error {
	rolePerms := models.S(models.TableIAMRolePermissions)
	roles := models.S(models.TableIAMRoles)
	var query string
	if strings.EqualFold(db.Dialector.Name(), "sqlite") {
		query = fmt.Sprintf(
			`UPDATE %[1]s SET tenant_uuid = (SELECT tenant_uuid FROM %[2]s WHERE %[2]s.id = %[1]s.role_id) WHERE tenant_uuid IS NULL OR trim(tenant_uuid) = ''`,
			rolePerms, roles,
		)
	} else {
		query = fmt.Sprintf(
			`UPDATE %s rp SET tenant_uuid = r.tenant_uuid FROM %s r WHERE rp.role_id = r.id AND (rp.tenant_uuid IS NULL OR rp.tenant_uuid::text = '')`,
			rolePerms, roles,
		)
	}
	if err := db.WithContext(ctx).Exec(query).Error; err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "tenant_uuid") && strings.Contains(lower, "column") {
			log.Printf("[migrate] tenant_uuid column missing on %s, skip backfill: %v", rolePerms, err)
			return nil
		}
		return err
	}
	return nil
}

func ResetDatabase(ctx context.Context, db *gorm.DB, cfg *config.DatabaseConfig) error {
	if strings.EqualFold(db.Dialector.Name(), "sqlite") || strings.TrimSpace(cfg.Schema) == "" {
		tables := append([]interface{}{}, businessTables...)
		tables = append(tables, iamTables...)
		return db.WithContext(ctx).Migrator().DropTable(tables...)
	}

	// 如果你用 GORM，可以直接 drop 所有表
	// 或者先获取表名，再循环 drop
	// 这里举例简单版本：
	err := db.Exec("DROP SCHEMA " + cfg.Schema + " CASCADE; CREATE SCHEMA " + cfg.Schema + ";").Error
	if err != nil {
		return err
	}
	return nil
}
