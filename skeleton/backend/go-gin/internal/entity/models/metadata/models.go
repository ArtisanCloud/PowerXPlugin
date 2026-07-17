package metadata

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DictionaryNamespace struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_dictionary_namespaces_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_metadata_dictionary_namespaces_tenant_namespace,priority:1;index" json:"tenant_uuid"`
	Namespace       string            `gorm:"column:namespace;type:varchar(160);not null;uniqueIndex:uk_metadata_dictionary_namespaces_tenant_namespace,priority:2;index" json:"namespace"`
	Module          string            `gorm:"column:module;type:varchar(96);not null;index" json:"module"`
	NameI18n        datatypes.JSONMap `gorm:"column:name_i18n;type:jsonb;default:'{}'::jsonb" json:"name_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
}

func (DictionaryNamespace) TableName() string {
	return models.S(models.TableMetadataDictionaryNamespaces)
}

func (m *DictionaryNamespace) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type DictionaryItem struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_dictionary_items_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	NamespaceUUID   string            `gorm:"column:namespace_uuid;type:uuid;not null;uniqueIndex:uk_metadata_dictionary_items_namespace_code,priority:1;index" json:"namespace_uuid"`
	Code            string            `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_metadata_dictionary_items_namespace_code,priority:2;index" json:"code"`
	LabelI18n       datatypes.JSONMap `gorm:"column:label_i18n;type:jsonb;default:'{}'::jsonb" json:"label_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	SortOrder       int               `gorm:"column:sort_order;not null;default:0;index" json:"sort_order"`
	ReferenceCount  int64             `gorm:"column:reference_count;not null;default:0" json:"reference_count"`
}

func (DictionaryItem) TableName() string { return models.S(models.TableMetadataDictionaryItems) }

func (m *DictionaryItem) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type Taxonomy struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_taxonomies_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_metadata_taxonomies_tenant_namespace,priority:1;index" json:"tenant_uuid"`
	Namespace       string            `gorm:"column:namespace;type:varchar(160);not null;uniqueIndex:uk_metadata_taxonomies_tenant_namespace,priority:2;index" json:"namespace"`
	Module          string            `gorm:"column:module;type:varchar(96);not null;index" json:"module"`
	NameI18n        datatypes.JSONMap `gorm:"column:name_i18n;type:jsonb;default:'{}'::jsonb" json:"name_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	MaxDepth        int               `gorm:"column:max_depth;not null;default:6" json:"max_depth"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
}

func (Taxonomy) TableName() string { return models.S(models.TableMetadataTaxonomies) }

func (m *Taxonomy) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type TaxonomyNode struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_taxonomy_nodes_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	TaxonomyUUID    string            `gorm:"column:taxonomy_uuid;type:uuid;not null;uniqueIndex:uk_metadata_taxonomy_nodes_taxonomy_code,priority:1;index" json:"taxonomy_uuid"`
	ParentUUID      *string           `gorm:"column:parent_uuid;type:uuid;index" json:"parent_uuid,omitempty"`
	Code            string            `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_metadata_taxonomy_nodes_taxonomy_code,priority:2;index" json:"code"`
	LabelI18n       datatypes.JSONMap `gorm:"column:label_i18n;type:jsonb;default:'{}'::jsonb" json:"label_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	Path            string            `gorm:"column:path;type:varchar(768);not null;index" json:"path"`
	Depth           int               `gorm:"column:depth;not null;default:0;index" json:"depth"`
	SortOrder       int               `gorm:"column:sort_order;not null;default:0;index" json:"sort_order"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	ReferenceCount  int64             `gorm:"column:reference_count;not null;default:0" json:"reference_count"`
	Version         int64             `gorm:"column:version;not null;default:1" json:"version"`
}

func (TaxonomyNode) TableName() string { return models.S(models.TableMetadataTaxonomyNodes) }

func (m *TaxonomyNode) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type Tag struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_tags_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_metadata_tags_tenant_namespace_resource_code,priority:1;index" json:"tenant_uuid"`
	Namespace       string            `gorm:"column:namespace;type:varchar(160);not null;uniqueIndex:uk_metadata_tags_tenant_namespace_resource_code,priority:2;index" json:"namespace"`
	ResourceType    string            `gorm:"column:resource_type;type:varchar(160);not null;uniqueIndex:uk_metadata_tags_tenant_namespace_resource_code,priority:3;index" json:"resource_type"`
	Code            string            `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_metadata_tags_tenant_namespace_resource_code,priority:4;index" json:"code"`
	LabelI18n       datatypes.JSONMap `gorm:"column:label_i18n;type:jsonb;default:'{}'::jsonb" json:"label_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	Color           string            `gorm:"column:color;type:varchar(32)" json:"color,omitempty"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
	UsageCount      int64             `gorm:"column:usage_count;not null;default:0" json:"usage_count"`
}

func (Tag) TableName() string { return models.S(models.TableMetadataTags) }

func (m *Tag) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}

type TagBinding struct {
	models.BaseNoTenantModel
	TenantUUID   string `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_metadata_tag_bindings_resource_tag,priority:1;index" json:"tenant_uuid"`
	ResourceType string `gorm:"column:resource_type;type:varchar(160);not null;uniqueIndex:uk_metadata_tag_bindings_resource_tag,priority:2;index" json:"resource_type"`
	ResourceUUID string `gorm:"column:resource_uuid;type:uuid;not null;uniqueIndex:uk_metadata_tag_bindings_resource_tag,priority:3;index" json:"resource_uuid"`
	TagUUID      string `gorm:"column:tag_uuid;type:uuid;not null;uniqueIndex:uk_metadata_tag_bindings_resource_tag,priority:4;index" json:"tag_uuid"`
}

func (TagBinding) TableName() string { return models.S(models.TableMetadataTagBindings) }

type ResourceType struct {
	models.BaseNoTenantModel
	UUID            string            `gorm:"column:uuid;type:uuid;not null;uniqueIndex:uk_metadata_resource_types_uuid" json:"uuid"`
	TenantUUID      string            `gorm:"column:tenant_uuid;type:uuid;not null;uniqueIndex:uk_metadata_resource_types_tenant_type,priority:1;index" json:"tenant_uuid"`
	ResourceType    string            `gorm:"column:resource_type;type:varchar(160);not null;uniqueIndex:uk_metadata_resource_types_tenant_type,priority:2;index" json:"resource_type"`
	Module          string            `gorm:"column:module;type:varchar(96);not null;index" json:"module"`
	NameI18n        datatypes.JSONMap `gorm:"column:name_i18n;type:jsonb;default:'{}'::jsonb" json:"name_i18n"`
	DescriptionI18n datatypes.JSONMap `gorm:"column:description_i18n;type:jsonb;default:'{}'::jsonb" json:"description_i18n,omitempty"`
	ValidatorKey    string            `gorm:"column:validator_key;type:varchar(160)" json:"validator_key,omitempty"`
	BindingEnabled  bool              `gorm:"column:binding_enabled;not null;default:true" json:"binding_enabled"`
	ValidatorStatus string            `gorm:"column:validator_status;type:varchar(32);not null;default:'ready'" json:"validator_status"`
	Status          string            `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"`
}

func (ResourceType) TableName() string { return models.S(models.TableMetadataResourceTypes) }

func (m *ResourceType) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = uuid.NewString()
	}
	return nil
}
