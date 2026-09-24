package migrate

import (
	"context"
	"fmt"
	"testing"

	EntityModels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	identitymodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// legacyLocalAISetting reproduces the profile schema released before the
// catalog source became a tenant system setting.
type legacyLocalAISetting struct {
	ID            uint64 `gorm:"primaryKey"`
	TenantUUID    string `gorm:"column:tenant_uuid"`
	CatalogSource string `gorm:"column:catalog_source"`
}

func (legacyLocalAISetting) TableName() string { return EntityModels.LocalAISetting{}.TableName() }

type legacySingleAIProfile struct {
	ID          uint64         `gorm:"primaryKey"`
	TenantUUID  string         `gorm:"column:tenant_uuid;uniqueIndex:uk_local_ai_settings_tenant_env_modality,priority:1"`
	Environment string         `gorm:"column:environment;uniqueIndex:uk_local_ai_settings_tenant_env_modality,priority:2"`
	Modality    string         `gorm:"column:modality;uniqueIndex:uk_local_ai_settings_tenant_env_modality,priority:3"`
	Provider    string         `gorm:"column:provider"`
	ModelKey    string         `gorm:"column:model_key"`
	Parameters  datatypes.JSON `gorm:"column:parameters;type:json;not null"`
}

// legacyLocalKnowledgeSpace reproduces the former space shape.  `name` and
// the JSON vector table are intentionally absent after the migration.
type legacyLocalKnowledgeSpace struct {
	ID                   uint64  `gorm:"primaryKey"`
	UUID                 string  `gorm:"type:uuid;not null;uniqueIndex"`
	TenantUUID           string  `gorm:"type:uuid;not null"`
	Name                 string  `gorm:"type:varchar(128);not null"`
	Description          string  `gorm:"type:text"`
	IngestionProfileUUID *string `gorm:"type:uuid"`
	IndexProfileUUID     *string `gorm:"type:uuid"`
	RAGProfileUUID       *string `gorm:"type:uuid"`
}

func (legacyLocalKnowledgeSpace) TableName() string {
	return EntityModels.LocalKnowledgeSpace{}.TableName()
}

func (legacySingleAIProfile) TableName() string { return EntityModels.LocalAISetting{}.TableName() }

type legacyLocalKnowledgeVectorIndex struct {
	ID        uint64 `gorm:"primaryKey"`
	UUID      string `gorm:"column:uuid"`
	SpaceUUID string `gorm:"column:space_uuid"`
	IndexKey  string `gorm:"column:index_key"`
}

func (legacyLocalKnowledgeVectorIndex) TableName() string {
	return EntityModels.LocalKnowledgeVectorIndex{}.TableName()
}

func TestMigratePluginModelsIncludesFederatedIAMTables(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() {
		EntityModels.ForceSchemaForTests("public")
	})

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}

	mustHave := []interface{}{
		&identitymodel.Role{},
		&identitymodel.Department{},
		&identitymodel.Permission{},
		&identitymodel.FederatedExternalIdentity{},
		&identitymodel.FederatedBinding{},
		&identitymodel.FederatedLoginChallenge{},
		&identitymodel.FederatedRiskEvent{},
	}
	for _, model := range mustHave {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("HasTable(%T) = false, want true", model)
		}
	}
	for _, table := range []string{identitymodel.Role{}.TableName(), identitymodel.Department{}.TableName(), identitymodel.Permission{}.TableName()} {
		if !db.Migrator().HasColumn(table, "uuid") {
			t.Fatalf("%s.uuid = missing", table)
		}
	}
	for table, column := range map[string]string{
		identitymodel.Member{}.TableName():         "department_uuid",
		identitymodel.Department{}.TableName():     "parent_department_uuid",
		identitymodel.MemberRole{}.TableName():     "member_uuid",
		identitymodel.RolePermission{}.TableName(): "permission_uuid",
	} {
		if !db.Migrator().HasColumn(table, column) {
			t.Fatalf("%s.%s = missing", table, column)
		}
	}
}

func TestEnsureContactIdentityChannelDictionaryItemUUIDPreservesLegacyRowsForExplicitRepair(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:legacy_contact_identity_" + uuid.NewString() + "?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE customer_contact_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, channel TEXT NOT NULL, external_subject TEXT NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO customer_contact_identities (channel, external_subject) VALUES ('email', 'legacy@example.test')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureContactIdentityChannelDictionaryItemUUID(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn("customer_contact_identities", "channel_dictionary_item_uuid") {
		t.Fatal("channel_dictionary_item_uuid is missing")
	}
	var assigned *string
	if err := db.Raw(`SELECT channel_dictionary_item_uuid FROM customer_contact_identities WHERE external_subject = ?`, "legacy@example.test").Scan(&assigned).Error; err != nil {
		t.Fatal(err)
	}
	if assigned != nil {
		t.Fatalf("legacy channel was inferred as %q", *assigned)
	}
}

func TestMigratePluginModelsIncludesCoreAlignedLocalKnowledgeTables(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:local_knowledge_schema_" + uuid.NewString() + "?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	models := []interface{}{
		&EntityModels.LocalKnowledgeSpace{}, &EntityModels.LocalKnowledgeDocument{}, &EntityModels.LocalKnowledgeChunk{}, &EntityModels.LocalKnowledgeKGNode{}, &EntityModels.LocalKnowledgeKGEdge{},
		&EntityModels.LocalIngestionProfileVersion{}, &EntityModels.LocalIndexProfileVersion{}, &EntityModels.LocalRAGProfileVersion{},
		&EntityModels.LocalKnowledgeIngestionJob{}, &EntityModels.LocalKnowledgeJobChunk{}, &EntityModels.LocalKnowledgeIndexJob{}, &EntityModels.LocalKnowledgeVectorIndex{},
		&EntityModels.LocalKnowledgeArtifactBundle{}, &EntityModels.LocalKnowledgeAuditTrailEntry{}, &EntityModels.LocalKnowledgeCorpusCheckJob{},
		&EntityModels.LocalKnowledgeDecayTask{}, &EntityModels.LocalKnowledgeDeltaJob{}, &EntityModels.LocalKnowledgeFeedbackCase{},
		&EntityModels.LocalKnowledgeFusionStrategyVersion{}, &EntityModels.LocalKnowledgeIAMSyncTask{},
		&EntityModels.LocalKnowledgePolicyTemplateVersion{}, &EntityModels.LocalKnowledgeSourceConnectorInstance{},
		&EntityModels.LocalKnowledgeSourceCredential{}, &EntityModels.LocalKnowledgeSpaceSyncJob{},
		&EntityModels.LocalKnowledgeReleasePolicy{}, &EntityModels.LocalKnowledgeReleaseBatch{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	for _, model := range models {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("HasTable(%T) = false", model)
		}
	}
	for table, column := range map[string]string{
		EntityModels.LocalKnowledgeSpace{}.TableName():        "policy_template_version_uuid",
		EntityModels.LocalKnowledgeDocument{}.TableName():     "index_status",
		EntityModels.LocalKnowledgeIngestionJob{}.TableName(): "artifact_bundle_uuid",
		EntityModels.LocalKnowledgeFeedbackCase{}.TableName(): "reprocess_job_uuid",
		EntityModels.LocalKnowledgeReleaseBatch{}.TableName(): "policy_uuid",
	} {
		if !db.Migrator().HasColumn(table, column) {
			t.Fatalf("%s.%s = missing", table, column)
		}
	}
	if !db.Migrator().HasColumn(EntityModels.LocalKnowledgeIngestionJob{}.TableName(), "progress_percent") {
		t.Fatal("local knowledge ingestion progress_percent column = missing")
	}
	spaceTable := EntityModels.LocalKnowledgeSpace{}.TableName()
	for _, column := range []string{"space_name", "ingestion_profile_key", "index_profile_key", "rag_profile_key", "embedding_profile_key", "active_vector_index_key"} {
		if !db.Migrator().HasColumn(spaceTable, column) {
			t.Fatalf("%s.%s = missing", spaceTable, column)
		}
	}
	if db.Migrator().HasColumn(spaceTable, "description") {
		t.Fatalf("%s.description = unexpected legacy plugin-only column", spaceTable)
	}
}

func TestMigratePluginModelsRemovesLegacyKnowledgeSpaceNameAndJSONVectors(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:legacy_knowledge_space_" + uuid.NewString() + "?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&legacyLocalKnowledgeSpace{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE local_knowledge_chunk_vectors (id integer primary key, vector text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&legacyLocalKnowledgeSpace{UUID: uuid.NewString(), TenantUUID: "11111111-1111-4111-8111-111111111111", Name: "legacy-space"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigratePluginModels(context.Background(), db, false); err != nil {
		t.Fatal(err)
	}
	spaceTable := EntityModels.LocalKnowledgeSpace{}.TableName()
	for _, column := range []string{"name", "description", "ingestion_profile_uuid", "index_profile_uuid", "rag_profile_uuid"} {
		hasLegacyColumn, err := hasPhysicalColumn(db, spaceTable, column)
		if err != nil {
			t.Fatal(err)
		}
		if hasLegacyColumn {
			t.Fatalf("%s.%s = legacy column was not removed", spaceTable, column)
		}
	}
	hasSpaceName, err := hasPhysicalColumn(db, spaceTable, "space_name")
	if err != nil {
		t.Fatal(err)
	}
	if !hasSpaceName {
		t.Fatalf("%s.space_name = missing", spaceTable)
	}
	var spaceName string
	if err := db.Raw("SELECT space_name FROM local_knowledge_spaces WHERE tenant_uuid = ?", "11111111-1111-4111-8111-111111111111").Scan(&spaceName).Error; err != nil {
		t.Fatal(err)
	}
	if spaceName != "legacy-space" {
		t.Fatalf("space_name = %q, want legacy-space", spaceName)
	}
	if db.Migrator().HasTable("local_knowledge_chunk_vectors") {
		t.Fatal("local_knowledge_chunk_vectors = legacy table was not removed")
	}
}

func TestRemoveObsoleteAICatalogSourceDeletesSavedPreference(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:ai_catalog_source_migration?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&legacyLocalAISetting{}, &EntityModels.PluginSystemConfig{}); err != nil {
		t.Fatal(err)
	}
	tenant := "11111111-1111-4111-8111-111111111111"
	if err := db.Create(&legacyLocalAISetting{TenantUUID: tenant, CatalogSource: "powerx"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&EntityModels.PluginSystemConfig{TenantUUID: tenant, Key: "ai.catalog_source", Value: []byte(`"powerx"`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := removeObsoleteAICatalogSource(context.Background(), db); err != nil {
		t.Fatalf("removeObsoleteAICatalogSource() error = %v", err)
	}
	if db.Migrator().HasColumn(EntityModels.LocalAISetting{}.TableName(), "catalog_source") {
		t.Fatal("legacy local_ai_settings.catalog_source still exists")
	}
	var count int64
	if err := db.Model(&EntityModels.PluginSystemConfig{}).Where("tenant_uuid = ? AND key = ?", tenant, "ai.catalog_source").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("obsolete ai.catalog_source count=%d, want 0", count)
	}
}

func TestMigrateLocalAISettingSourcesSplitsLegacyProfileSlot(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:ai_profile_source_migration?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&legacySingleAIProfile{}); err != nil {
		t.Fatal(err)
	}
	legacy := legacySingleAIProfile{TenantUUID: "11111111-1111-4111-8111-111111111111", Environment: "development", Modality: "llm", Provider: "baidu", ModelKey: "ERNIE-Bot-4", Parameters: datatypes.JSON([]byte(`{}`))}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE local_ai_settings ADD COLUMN source text NOT NULL DEFAULT 'local'").Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateLocalAISettingSources(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var local struct {
		Provider string `gorm:"column:provider"`
		ModelKey string `gorm:"column:model_key"`
		Source   string `gorm:"column:source"`
	}
	if err := db.Table(EntityModels.LocalAISetting{}.TableName()).Select("provider, model_key, source").Where("tenant_uuid = ? AND environment = ? AND modality = ? AND source = ?", legacy.TenantUUID, "development", "llm", "local").Take(&local).Error; err != nil {
		t.Fatal(err)
	}
	if local.Provider != "baidu" || local.ModelKey != "ERNIE-Bot-4" {
		t.Fatalf("migrated profile=%+v", local)
	}
	if err := db.Exec("INSERT INTO local_ai_settings (tenant_uuid, environment, modality, source, provider, model_key, parameters) VALUES (?, ?, ?, ?, ?, ?, ?)", legacy.TenantUUID, "development", "llm", "powerx", "ollama", "qwen3:8b", "{}").Error; err != nil {
		t.Fatalf("create powerx partition: %v", err)
	}
	if db.Migrator().HasIndex(&EntityModels.LocalAISetting{}, "uk_local_ai_settings_tenant_env_modality") {
		t.Fatal("legacy unique index still exists")
	}
	if !db.Migrator().HasIndex(&EntityModels.LocalAISetting{}, "uk_local_ai_settings_tenant_env_modality_source") {
		t.Fatal("source-partitioned unique index is missing")
	}
}

func TestBackfillIAMUUIDRelations(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam_uuid_backfill?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := MigratePluginModels(context.Background(), db, true); err != nil {
		t.Fatalf("MigratePluginModels() error = %v", err)
	}
	tenant := "tenant-uuid-backfill"
	parent := identitymodel.Department{}
	parent.TenantUuid, parent.Name, parent.Code, parent.Path = tenant, "Parent", "parent", "parent"
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := identitymodel.Department{}
	child.TenantUuid, child.Name, child.Code, child.Path = tenant, "Child", "child", "parent.child"
	child.ParentID = &parent.ID
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	account := identitymodel.User{Email: "backfill@example.test", DisplayName: "Backfill", PasswordHash: "hash"}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	member := identitymodel.Member{UserID: account.ID, Username: "backfill", Status: identitymodel.StatusActive, DepartmentID: &child.ID}
	member.TenantUuid = tenant
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	role := identitymodel.Role{Code: "backfill-role", Name: "Backfill role", ScopeType: identitymodel.RoleScopeTenant}
	role.TenantUuid = tenant
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	permission := identitymodel.Permission{Description: "Backfill permission", Resource: "backfill", Action: "read"}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatal(err)
	}
	memberRole := identitymodel.MemberRole{UserID: member.ID, RoleID: role.ID}
	if err := db.Create(&memberRole).Error; err != nil {
		t.Fatal(err)
	}
	rolePermission := identitymodel.RolePermission{RoleID: role.ID, PermissionID: permission.ID}
	if err := db.Create(&rolePermission).Error; err != nil {
		t.Fatal(err)
	}
	// Simulate rows created before UUID columns existed.
	for index, entry := range []struct {
		table string
		id    uint64
	}{
		{identitymodel.Department{}.TableName(), parent.ID}, {identitymodel.Department{}.TableName(), child.ID},
		{identitymodel.Member{}.TableName(), member.ID}, {identitymodel.Role{}.TableName(), role.ID},
		{identitymodel.Permission{}.TableName(), permission.ID},
	} {
		if err := db.Table(entry.table).Where("id = ?", entry.id).Update("uuid", fmt.Sprintf("legacy-%d", index)).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := backfillIAMUUIDRelations(context.Background(), db); err != nil {
		t.Fatalf("backfillIAMUUIDRelations() error = %v", err)
	}
	var gotChild identitymodel.Department
	if err := db.First(&gotChild, child.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotParent identitymodel.Department
	if err := db.First(&gotParent, parent.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotChild.UUID == "" || gotParent.UUID == "" || gotChild.ParentDepartmentUUID == nil || *gotChild.ParentDepartmentUUID != gotParent.UUID {
		t.Fatalf("parent uuid relation was not backfilled: child=%#v parent=%#v", gotChild, gotParent)
	}
	var gotMember identitymodel.Member
	if err := db.First(&gotMember, member.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotMember.UUID == "" || gotMember.DepartmentUUID == nil || *gotMember.DepartmentUUID != gotChild.UUID {
		t.Fatalf("department uuid relation was not backfilled: %#v", gotMember)
	}
	var gotMemberRole identitymodel.MemberRole
	if err := db.First(&gotMemberRole, "member_id = ? AND role_id = ?", member.ID, role.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotRole identitymodel.Role
	if err := db.First(&gotRole, role.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotRole.UUID == "" || gotMemberRole.MemberUUID != gotMember.UUID || gotMemberRole.RoleUUID != gotRole.UUID {
		t.Fatalf("member role uuid relation not backfilled: %#v", gotMemberRole)
	}
	var gotRolePermission identitymodel.RolePermission
	if err := db.First(&gotRolePermission, "role_id = ? AND permission_id = ?", role.ID, permission.ID).Error; err != nil {
		t.Fatal(err)
	}
	var gotPermission identitymodel.Permission
	if err := db.First(&gotPermission, permission.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotPermission.UUID == "" || gotRolePermission.RoleUUID != gotRole.UUID || gotRolePermission.PermissionUUID != gotPermission.UUID {
		t.Fatalf("role permission uuid relation not backfilled: %#v", gotRolePermission)
	}
}

func TestEnsureIAMIdentityUUIDsBackfillsBeforeAutoMigrate(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam_pre_migrate_uuid?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tables := []string{
		identitymodel.User{}.TableName(), identitymodel.Member{}.TableName(), identitymodel.Role{}.TableName(),
		identitymodel.Department{}.TableName(), identitymodel.Permission{}.TableName(),
	}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, uuid TEXT NULL)", table)).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(fmt.Sprintf("INSERT INTO %s (id, uuid) VALUES (1, NULL)", table)).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := ensureIAMIdentityUUIDs(context.Background(), db); err != nil {
		t.Fatalf("ensureIAMIdentityUUIDs() error = %v", err)
	}
	for _, table := range tables {
		var count int64
		if err := db.Table(table).Where("uuid IS NULL OR trim(uuid) = ''").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d empty uuid rows", table, count)
		}
	}
}

func TestEnsureLocalKnowledgeVectorIndexTenantBackfillsOwningSpace(t *testing.T) {
	EntityModels.ForceSchemaForTests("")
	t.Cleanup(func() { EntityModels.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:knowledge_vector_tenant?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&EntityModels.LocalKnowledgeSpace{}, &legacyLocalKnowledgeVectorIndex{}); err != nil {
		t.Fatal(err)
	}
	space := EntityModels.LocalKnowledgeSpace{TenantUUID: "11111111-1111-4111-8111-111111111111", SpaceName: "support", DepartmentCode: "support"}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&legacyLocalKnowledgeVectorIndex{UUID: "22222222-2222-4222-8222-222222222222", SpaceUUID: space.UUID, IndexKey: "dense"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureLocalKnowledgeVectorIndexTenant(context.Background(), db); err != nil {
		t.Fatalf("ensureLocalKnowledgeVectorIndexTenant() error = %v", err)
	}
	var got struct {
		TenantUUID string `gorm:"column:tenant_uuid"`
	}
	if err := db.Table(EntityModels.LocalKnowledgeVectorIndex{}.TableName()).Select("tenant_uuid").Where("uuid = ?", "22222222-2222-4222-8222-222222222222").Take(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.TenantUUID != space.TenantUUID {
		t.Fatalf("tenant uuid=%q want %q", got.TenantUUID, space.TenantUUID)
	}
}
